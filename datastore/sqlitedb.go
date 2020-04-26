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
	"cod/util"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var CurrentSchemaVersion = 1

type Storage interface {
	GetCommandPolicy(args []string) (policy Policy, err error)

	GetAllCompletions() (pages []HelpPage, err error)
	GetCompletions(path string) (completions []Completion, err error)

	AddHelpPage(helpPage *HelpPage, policy Policy) (status AddHelpPageStatus, err error)

	// NB. This command might return null pointers in case some help page is broken.
	ListCommands() (result map[int64]*Command, err error)

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

func getCompletionsForExecutable(tx *sql.Tx, executablePath string) (completions []Completion, err error) {
	completionRows, err := tx.Query(`
				select Completion.Flag, Completion.Context
				from Completion inner join HelpPage on Completion.HelpPageId = HelpPage.HelpPageId
				where HelpPage.ExecutablePath = ?
			`, executablePath)
	if err != nil {
		return
	}
	defer func() {
		_ = completionRows.Close()
	}()

	for completionRows.Next() {
		var contextBytes sql.NullString
		completion := Completion{}
		err = completionRows.Scan(&completion.Flag, &contextBytes)
		util.VerifyPanic(err)
		if contextBytes.Valid {
			err = json.Unmarshal([]byte(contextBytes.String), &completion.Context)
			if err != nil {
				return
			}
		}
		completions = append(completions, completion)
	}
	err = completionRows.Err()
	return
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

func (s *sqliteStorage) RemoveHelpPage(helpPageId int64) (executablePath string, err error) {
	err = withTransaction(s.db, func(tx *sql.Tx) (err error) {
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
		select HelpPage.ExecutablePath, Completion.Flag
		from Completion
		inner join HelpPage on (HelpPage.HelpPageId = Completion.HelpPageId)
	`)

	if err != nil {
		return
	}

	helpPageMap := make(map[string]*HelpPage)
	for rows.Next() {
		var executablePath, flag string
		err = rows.Scan(&executablePath, &flag)
		if err != nil {
			return
		}

		helpPage, ok := helpPageMap[executablePath]
		if !ok {
			helpPage = &HelpPage{
				ExecutablePath: executablePath,
			}
			helpPageMap[executablePath] = helpPage
		}
		helpPage.Completions = append(
			helpPage.Completions,
			Completion{
				Flag: flag,
			},
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
		insert into Completion(HelpPageId, Flag, Context) values (?, ?, ?)
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
		_, err = completionStatement.Exec(helpPageId, completion.Flag, contextBytes)
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

	if err == nil {
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
	} else if err == sql.ErrNoRows {
		err = nil
	} else {
		return
	}

	err = tx.QueryRow(`
			select HelpPageId from HelpPage where ExecutablePath = ? and CommandArgsCheckSum = ?
			`, executablePath, commandCheckSum,
	).Scan(&helpPageId)

	if err == nil {
		if rowIdToReuse == nil || *rowIdToReuse > helpPageId {
			var tmp = helpPageId
			rowIdToReuse = &tmp
		}
		err = removeHelpPage(tx, helpPageId)
		if err != nil {
			return
		}
	} else if err == sql.ErrNoRows {
		err = nil
	} else {
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
			                     Policy
			) values (?, ?, ?, ?, ?, ?)
		`, rowIdToReplace,
		executablePath,
		helpPage.CheckSum,
		commandChecksum,
		helpPageCommandJson,
		policy)
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

	_, err = db.Exec(`
		begin;

		create table Completion (
			CompletionId   integer not null primary key autoincrement,
			HelpPageId     integer not null,
			Flag           text not null,
			Context        text,
			foreign key (HelpPageId) references HelpPage(HelpPageId)
		);
		create index Completion_SourceId ON Completion (HelpPageId);

		create table HelpPage (
		    HelpPageId          integer not null primary key autoincrement,
			ExecutablePath      text,
			HelpTextCheckSum    text,
			CommandArgsCheckSum text,
			CommandJson         text,
			Policy              text,
			unique              (ExecutablePath, HelpTextCheckSum),
			unique              (ExecutablePath, CommandArgsCheckSum)
		);
		create index HelpPage_ExecutablePath ON HelpPage (ExecutablePath);
		create index HelpPage_ExecutablePath_HelpTextCheckSum ON HelpPage (ExecutablePath, HelpTextCheckSum);
		create index HelpPage_ExecutablePath_CommandArgsCheckSum ON HelpPage (ExecutablePath, CommandArgsCheckSum);

		PRAGMA user_version = 1;

		commit;
	`, userVersion)
	return
}

func migrateSchema(userVersion int, _ *sql.DB) (err error) {
	switch userVersion {
	case CurrentSchemaVersion:
		break
	default:
		panic(fmt.Errorf("unknown db version: %v", userVersion))
	}
	return
}
