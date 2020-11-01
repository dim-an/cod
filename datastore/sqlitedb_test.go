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
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/dim-an/cod/util"
	"github.com/stretchr/testify/require"
)

type testSqliteStorage struct {
	Storage
	tempfile string
	t        *testing.T
}

func (s *testSqliteStorage) CheckedClose() {
	err := s.Storage.Close()
	require.Nil(s.t, err)

	err = os.Remove(s.tempfile)
	util.VerifyPanic(err)
}

func newTestSqliteStorage(t *testing.T) Storage {
	tmp, err := ioutil.TempFile("", "cod-sqlite")
	util.VerifyPanic(err)

	filename := tmp.Name()
	err = tmp.Close()

	util.VerifyPanic(err)
	db, err := NewSqliteStorage(filename)
	require.Nil(t, err)

	return &testSqliteStorage{
		Storage:  db,
		tempfile: filename,
		t:        t,
	}

}

func TestNewSqliteStorage(t *testing.T) {
	db := newTestSqliteStorage(t)
	err := db.Close()
	require.Nil(t, err)
}

func TestCRUD(t *testing.T) {
	db := newTestSqliteStorage(t)

	status, err := db.AddHelpPage(
		&HelpPage{
			ExecutablePath: "/my-test-command",
			Completions: []Completion{
				{Flag: "foo"},
				{Flag: "bar"},
				{Flag: "baz"},
				{Flag: "qux"},
			},
			CheckSum: "100500",
		},
		PolicyUnknown,
	)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	status, err = db.AddHelpPage(
		&HelpPage{
			ExecutablePath: "/my-test-command-2",
			Completions: []Completion{
				{Flag: "x1"},
				{Flag: "y1"},
				{Flag: "y2"},
				{Flag: "y3"},
			},
			CheckSum: "42",
		},
		PolicyUnknown,
	)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	items, err := db.GetCompletions("/my-test-command")
	require.Nil(t, err)
	require.Equal(t,
		[]Completion{
			{Flag: "foo"},
			{Flag: "bar"},
			{Flag: "baz"},
			{Flag: "qux"},
		},
		items,
	)
}

func TestAddSamePage(t *testing.T) {
	db := newTestSqliteStorage(t)

	helpPage := HelpPage{
		ExecutablePath: "/my-test-command",
		Completions: []Completion{
			{Flag: "foo"},
			{Flag: "bar"},
			{Flag: "baz"},
			{Flag: "qux"},
		},
		CheckSum: "100500",
	}

	status, err := db.AddHelpPage(&helpPage, PolicyUnknown)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	status, err = db.AddHelpPage(&helpPage, PolicyUnknown)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusUpdated)
}

func TestListCommands(t *testing.T) {
	db := newTestSqliteStorage(t)

	makeHelpPage := func(c *Command) *HelpPage {
		s := sha1.New()
		_, err := s.Write([]byte(c.Dir))
		util.VerifyPanic(err)
		for _, arg := range c.Args {
			_, err := s.Write([]byte(arg))
			util.VerifyPanic(err)
		}
		for _, arg := range c.Env {
			_, err = s.Write([]byte(arg))
			util.VerifyPanic(err)
		}

		return &HelpPage{
			ExecutablePath: c.Args[0],
			Completions:    nil,
			CheckSum:       fmt.Sprintf("%x", s.Sum(nil)),
			Command:        *c,
		}
	}

	command1 := Command{
		Args: []string{"/foo", "--help"},
		Dir:  "/tmp",
	}

	command2 := Command{
		Args: []string{"/bar", "--help"},
		Dir:  "/tmp",
	}

	command3 := Command{
		Args: []string{"/foo", "mode", "--help"},
		Dir:  "/tmp",
	}

	status, err := db.AddHelpPage(makeHelpPage(&command1), PolicyUnknown)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	status, err = db.AddHelpPage(makeHelpPage(&command2), PolicyUnknown)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	status, err = db.AddHelpPage(makeHelpPage(&command3), PolicyUnknown)
	require.Nil(t, err)
	require.Equal(t, status, AddHelpPageStatusNew)

	commandMap, err := db.ListCommands()
	require.Nil(t, err)

	toValues := func(m *map[int64]*Command) []*Command {
		var res []*Command
		for _, v := range *m {
			res = append(res, v)
		}
		sort.Sort(SortableCommands(res))
		return res
	}

	require.Equal(t, []*Command{&command2, &command1, &command3}, toValues(&commandMap))
}

type SortableCommands []*Command

func (cs SortableCommands) Len() int {
	return len(cs)
}

func cmpStringSlice(lhs, rhs []string) int {
	limit := len(lhs)
	if limit > len(rhs) {
		limit = len(rhs)
	}
	for i := 0; i < limit; i += 1 {
		switch strings.Compare(lhs[i], rhs[i]) {
		case 0:
			continue
		case 1:
			return 1
		case -1:
			return -1
		}
	}
	switch {
	case len(lhs) < len(rhs):
		return -1
	case len(lhs) > len(rhs):
		return 1
	default:
		return 0
	}
}

func (cs SortableCommands) Less(i, j int) bool {
	lhs := cs[i]
	rhs := cs[j]

	switch cmpStringSlice(lhs.Args, rhs.Args) {
	case -1:
		return true
	case 1:
		return false
	}

	switch cmpStringSlice(lhs.Env, rhs.Env) {
	case -1:
		return true
	case 1:
		return false
	}

	return lhs.Dir < rhs.Dir
}

func (cs SortableCommands) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}
