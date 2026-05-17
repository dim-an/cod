// Copyright 2020 Dmitry Ermolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dim-an/cod/util"
	_ "github.com/ncruces/go-sqlite3/driver"
)

var CurrentSchemaVersion = 2

type Storage interface {
	GetCommandPolicy(args []string) (policy Policy, err error)

	GetAllCompletions() (pages []HelpPage, err error)
	GetCompletions(path string) (completions []Completion, err error)
	GetCompletionsByPrefix(path, prefix string) (completions []Completion, err error)

	AddHelpPage(helpPage *HelpPage, policy Policy) (status AddHelpPageStatus, err error)

	// NB. This command might return null pointers in case some help page is broken.
	ListCommands() (result map[int64]*Command, err error)
	ListHelpPages() (result []HelpPageInfo, err error)

	RemoveHelpPage(commandId int64) (path string, err error)

	Close() error
}

func NewSqliteStorage(fileName string) (storage Storage, err error) {
	db, err := sql.Open("sqlite3", fileName)
	if err != nil {
		return
	}
	err = db.Ping()
	if err != nil {
		return
	}

	err = updateSchema(db)
	if err != nil {
		return
	}

	var foreignKey string
	err = db.QueryRow("pragma foreign_keys;").Scan(&foreignKey)
	if err != nil {
		return
	}
	if foreignKey == "" {
		err = fmt.Errorf("sqlite doesn't support foreign keys")
		return
	}
	_, err = db.Exec("pragma foreign_keys = on;")
	if err != nil {
		return
	}

	storage = &sqliteStorage{
		db: db,
	}
	return
}

func withTransaction(db *sql.DB, f func(tx *sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	err = f(tx)
	if err != nil {
		err2 := tx.Rollback()
		if err2 != nil {
			log.Printf("error occurred while transaction rollback: %v", err2)
		}
		return
	}
	err = tx.Commit()
	return
}

func scanCompletions(rows *sql.Rows) (completions []Completion, err error) {
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var contextBytes sql.NullString
		completion := Completion{}
		err = rows.Scan(&completion.Flag, &contextBytes, &completion.Description)
		util.VerifyPanic(err)
		if contextBytes.Valid {
			err = json.Unmarshal([]byte(contextBytes.String), &completion.Context)
			if err != nil {
				return
			}
		}
		completions = append(completions, completion)
	}
	err = rows.Err()
	return
}

func getCompletionsForExecutable(tx *sql.Tx, executablePath string) (completions []Completion, err error) {
	completionRows, err := tx.Query(`
				select Completion.Flag, Completion.Context, Completion.Description
				from Completion inner join HelpPage on Completion.HelpPageId = HelpPage.HelpPageId
				where HelpPage.ExecutablePath = ?
			`, executablePath)
	if err != nil {
		return
	}
	return scanCompletions(completionRows)
}

func getCompletionsForHelpPageId(tx *sql.Tx, helpPageId int64) (completions []Completion, err error) {
	completionRows, err := tx.Query(`
				select Completion.Flag, Completion.Context, Completion.Description
				from Completion
				where Completion.HelpPageId = ?
			`, helpPageId)
	if err != nil {
		return
	}
	return scanCompletions(completionRows)
}

func (s *sqliteStorage) GetCompletions(executablePath string) (completions []Completion, err error) {
	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
		var cur []Completion
		cur, err = getCompletionsForExecutable(tx, executablePath)
		if err != nil {
			return
		}
		completions = append(completions, cur...)
		return
	})
	return
}

func escapeLikePrefix(prefix string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(prefix) + "%"
}

func (s *sqliteStorage) GetCompletionsByPrefix(executablePath, prefix string) (completions []Completion, err error) {
	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
		rows, err := tx.Query(`
				select Completion.Flag, Completion.Context, Completion.Description
				from Completion inner join HelpPage on Completion.HelpPageId = HelpPage.HelpPageId
				where HelpPage.ExecutablePath = ? and Completion.Flag like ? escape '\'
			`, executablePath, escapeLikePrefix(prefix))
		if err != nil {
			return
		}
		completions, err = scanCompletions(rows)
		return
	})
	return
}

func (s *sqliteStorage) AddHelpPage(helpPage *HelpPage, policy Policy) (status AddHelpPageStatus, err error) {
	err = CheckHelpPage(helpPage)
	if err != nil {
		return
	}

	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
		commandChecksum := util.HashStrings(helpPage.Command.Args)

		if policy == PolicyUnknown {
			checkSum := util.HashStrings(helpPage.Command.Args)
			err = s.db.QueryRow(
				`select Policy from HelpPage where CommandArgsCheckSum = ?`,
				checkSum,
			).Scan(&policy)
			if err == sql.ErrNoRows {
				err = nil
				policy = PolicyUnknown
			} else if err != nil {
				return
			}
		}

		var rowIdToReplace *int64
		rowIdToReplace, err = removeAndMergeConflicting(tx, helpPage.ExecutablePath, commandChecksum, helpPage)
		if err != nil {
			return
		}

		err = insertHelpPage(tx, helpPage.ExecutablePath, rowIdToReplace, commandChecksum, helpPage, policy)
		if err != nil {
			return
		}

		if rowIdToReplace == nil {
			status = AddHelpPageStatusNew
		} else {
			status = AddHelpPageStatusUpdated
		}

		return
	})
	return
}

func (s *sqliteStorage) ListCommands() (result map[int64]*Command, err error) {
	rows, err := s.db.Query(`
		select HelpPageId, CommandJson from HelpPage
	`)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	result = make(map[int64]*Command)
	for rows.Next() {
		var helpPageId int64
		var commandJson []byte

		err = rows.Scan(&helpPageId, &commandJson)
		if err != nil {
			return
		}

		var command Command
		err = json.Unmarshal(commandJson, &command)
		if err != nil {
			log.Printf("HelpPage %v has broken commandJson field: %v", helpPageId, err)
			result[helpPageId] = nil
		} else {
			result[helpPageId] = &command
		}
	}
	err = rows.Err()
	return
}

func (s *sqliteStorage) ListHelpPages() (result []HelpPageInfo, err error) {
	var items []HelpPageInfo
	rows, err := s.db.Query(`
		select
			HelpPage.HelpPageId,
			HelpPage.ExecutablePath,
			HelpPage.Description,
			HelpPage.CommandJson,
			count(Completion.CompletionId)
		from HelpPage
		left join Completion on Completion.HelpPageId = HelpPage.HelpPageId
		group by HelpPage.HelpPageId
	`)
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var commandJson []byte
		item := HelpPageInfo{}
		err = rows.Scan(&item.Id, &item.ExecutablePath, &item.Description, &commandJson, &item.CompletionCount)
		if err != nil {
			return
		}

		var command Command
		err = json.Unmarshal(commandJson, &command)
		if err != nil {
			log.Printf("HelpPage %v has broken commandJson field: %v", item.Id, err)
		} else {
			item.Command = &command
		}

		items = append(items, item)
	}
	err = rows.Err()
	if err != nil {
		return
	}

	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
		for idx := range items {
			items[idx].Completions, err = getCompletionsForHelpPageId(tx, items[idx].Id)
			if err != nil {
				return
			}
		}
		return
	})
	result = items
	return
}

func (s *sqliteStorage) RemoveHelpPage(helpPageId int64) (executablePath string, err error) {
	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
		err = tx.QueryRow(
			`select ExecutablePath from HelpPage where HelpPageId = ?`,
			helpPageId,
		).Scan(&executablePath)
		if err != nil {
			return
		}
		err = removeHelpPage(tx, helpPageId)
		if err != nil {
			return
		}
		return
	})
	return
}

func (s *sqliteStorage) GetCommandPolicy(args []string) (policy Policy, err error) {
	checkSum := util.HashStrings(args)
	err = s.db.QueryRow(`select Policy from HelpPage where CommandArgsCheckSum = ?`, checkSum).Scan(&policy)
	if err == sql.ErrNoRows {
		err = nil
		policy = PolicyUnknown
	} else if err != nil {
		return
	}
	return
}

func (s *sqliteStorage) GetAllCompletions() (pages []HelpPage, err error) {
	rows, err := s.db.Query(`
		select HelpPage.ExecutablePath, HelpPage.Description, Completion.Flag, Completion.Context, Completion.Description
		from Completion
		inner join HelpPage on (HelpPage.HelpPageId = Completion.HelpPageId)
	`)

	if err != nil {
		return
	}

	helpPageMap := make(map[string]*HelpPage)
	for rows.Next() {
		var executablePath, helpPageDescription string
		var contextBytes sql.NullString
		completion := Completion{}
		err = rows.Scan(&executablePath, &helpPageDescription, &completion.Flag, &contextBytes, &completion.Description)
		if err != nil {
			return
		}
		if contextBytes.Valid {
			err = json.Unmarshal([]byte(contextBytes.String), &completion.Context)
			if err != nil {
				return
			}
		}

		helpPage, ok := helpPageMap[executablePath]
		if !ok {
			helpPage = &HelpPage{
				ExecutablePath: executablePath,
				Description:    helpPageDescription,
			}
			helpPageMap[executablePath] = helpPage
		}
		helpPage.Completions = append(
			helpPage.Completions,
			completion,
		)
	}
	err = rows.Err()
	if err != nil {
		return
	}

	for _, p := range helpPageMap {
		pages = append(pages, *p)
	}

	return
}
func (s *sqliteStorage) Close() error {
	return s.db.Close()
}

type sqliteStorage struct {
	db *sql.DB
}

func insertCompletions(tx *sql.Tx, helpPageId int64, completions []Completion) (err error) {
	completionStatement, err := tx.Prepare(`
		insert into Completion(HelpPageId, Flag, Context, Description) values (?, ?, ?, ?)
	`)
	if err != nil {
		return
	}

	for _, completion := range completions {
		var contextBytes []byte
		contextBytes, err = json.Marshal(completion.Context)
		if err != nil {
			return
		}
		_, err = completionStatement.Exec(helpPageId, completion.Flag, contextBytes, completion.Description)
		if err != nil {
			return
		}
	}
	return
}

func commandToJson(command Command) (s string, err error) {
	helpPageCommandJsonBytes, err := json.Marshal(command)
	if err != nil {
		return
	}
	s = string(helpPageCommandJsonBytes)
	return
}

func removeAndMergeConflicting(tx *sql.Tx, executablePath string, commandCheckSum string, helpPage *HelpPage) (rowIdToReuse *int64, err error) {
	var helpPageId int64
	var oldCommandJson string
	err = tx.QueryRow(`
			select HelpPageId, CommandJson from HelpPage where ExecutablePath = ? and HelpTextCheckSum = ?
			`, executablePath, helpPage.CheckSum,
	).Scan(&helpPageId, &oldCommandJson)
	// FIX QF1002
	switch err {
	case nil:
		var curCommandJson string
		curCommandJson, err = commandToJson(helpPage.Command)
		if err != nil {
			return
		}

		var oldCommand Command
		err = json.Unmarshal([]byte(oldCommandJson), &oldCommand)
		if err != nil {
			return
		}

		if len(curCommandJson) > len(oldCommandJson) {
			helpPage.Command = oldCommand
		}
		var tmp = helpPageId
		rowIdToReuse = &tmp
		err = removeHelpPage(tx, helpPageId)
		if err != nil {
			return
		}
	case sql.ErrNoRows:
		err = nil
	default:
		return
	}

	err = tx.QueryRow(`
			select HelpPageId from HelpPage where ExecutablePath = ? and CommandArgsCheckSum = ?
			`, executablePath, commandCheckSum,
	).Scan(&helpPageId)
	// FIX QF1002
	switch err {
	case nil:
		if rowIdToReuse == nil || *rowIdToReuse > helpPageId {
			var tmp = helpPageId
			rowIdToReuse = &tmp
		}
		err = removeHelpPage(tx, helpPageId)
		if err != nil {
			return
		}
	case sql.ErrNoRows:
		err = nil
	default:
		return
	}

	return
}

func insertHelpPage(tx *sql.Tx, executablePath string, rowIdToReplace *int64, commandChecksum string, helpPage *HelpPage, policy Policy) (err error) {
	helpPageCommandJson, err := commandToJson(helpPage.Command)
	if err != nil {
		return
	}

	res, err := tx.Exec(`
			insert into HelpPage(
			                     HelpPageId,
			                     ExecutablePath,
			                     HelpTextCheckSum,
			                     CommandArgsCheckSum,
			                     CommandJson,
			                     Policy,
			                     Description
			) values (?, ?, ?, ?, ?, ?, ?)
		`, rowIdToReplace,
		executablePath,
		helpPage.CheckSum,
		commandChecksum,
		helpPageCommandJson,
		policy,
		helpPage.Description)
	if err != nil {
		return
	}
	var helpPageId int64
	helpPageId, err = res.LastInsertId()
	if err != nil {
		return
	}

	err = insertCompletions(tx, helpPageId, helpPage.Completions)
	if err != nil {
		return
	}

	return
}

func removeHelpPage(tx *sql.Tx, helpPageId int64) (err error) {
	_, err = tx.Exec("delete from Completion where HelpPageId = ?", helpPageId)
	if err != nil {
		return
	}

	_, err = tx.Exec("delete from HelpPage where HelpPageId = ?", helpPageId)
	if err != nil {
		return
	}
	return
}

func updateSchema(db *sql.DB) (err error) {
	userVersion := -1
	err = db.QueryRow("PRAGMA user_version;").Scan(&userVersion)
	if err != nil {
		return
	}

	if userVersion != 0 {
		err = migrateSchema(userVersion, db)
		return
	}

	schemaStatements := []string{
		`create table Completion (
			CompletionId   integer not null primary key autoincrement,
			HelpPageId     integer not null,
			Flag           text not null,
			Context        text,
			Description    text not null default '',
			foreign key (HelpPageId) references HelpPage(HelpPageId)
		)`,
		`create index Completion_SourceId ON Completion (HelpPageId)`,
		`create index Completion_SourceId_Flag ON Completion (HelpPageId, Flag)`,
		`create table HelpPage (
		    HelpPageId          integer not null primary key autoincrement,
			ExecutablePath      text,
			HelpTextCheckSum    text,
			CommandArgsCheckSum text,
			CommandJson         text,
			Policy              text,
			Description         text not null default '',
			unique              (ExecutablePath, HelpTextCheckSum),
			unique              (ExecutablePath, CommandArgsCheckSum)
		)`,
		`create index HelpPage_ExecutablePath ON HelpPage (ExecutablePath)`,
		`create index HelpPage_ExecutablePath_HelpTextCheckSum ON HelpPage (ExecutablePath, HelpTextCheckSum)`,
		`create index HelpPage_ExecutablePath_CommandArgsCheckSum ON HelpPage (ExecutablePath, CommandArgsCheckSum)`,
		`PRAGMA user_version = 2`,
	}
	err = withTransaction(db, func(tx *sql.Tx) error {
		for _, stmt := range schemaStatements {
			if _, err := tx.Exec(stmt); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func migrateSchema(userVersion int, db *sql.DB) (err error) {
	switch userVersion {
	case CurrentSchemaVersion:
		break
	case 1:
		err = withTransaction(db, func(tx *sql.Tx) error {
			statements := []string{
				`alter table HelpPage add column Description text not null default ''`,
				`alter table Completion add column Description text not null default ''`,
				`create index Completion_SourceId_Flag ON Completion (HelpPageId, Flag)`,
				`PRAGMA user_version = 2`,
			}
			for _, stmt := range statements {
				if _, err := tx.Exec(stmt); err != nil {
					return err
				}
			}
			return nil
		})
	default:
		panic(fmt.Errorf("unknown db version: %v", userVersion))
	}
	return
}
