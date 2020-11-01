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

package test

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLearnList(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	out := wb.RunCodCmd("list")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/cat.py --help",
			"binaries/naval-fate.py --help",
		},
		commands,
	)
}

func TestLearnListRemove(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	out := wb.RunCodCmd("list")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/cat.py --help",
			"binaries/naval-fate.py --help",
		},
		commands,
	)

	wb.RunCodCmd("remove", "1")
	out = wb.RunCodCmd("list")
	commands = wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/naval-fate.py --help",
		},
		commands,
	)

	wb.RunCodCmd("remove", "2")
	out = wb.RunCodCmd("list")
	require.Equal(t, "", out)
}

func TestLearnListBasenameSelector(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	out := wb.RunCodCmd("list", "cat.py")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/cat.py --help",
		},
		commands,
	)
}

func TestLearnListPathSelector(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	abs, err := filepath.Abs("binaries/naval-fate.py")
	require.Nil(t, err)

	out := wb.RunCodCmd("list", abs)

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/naval-fate.py --help",
		},
		commands,
	)
}

func TestLearnListIdSelector(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	out := wb.RunCodCmd("list", "2")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/naval-fate.py --help",
		},
		commands,
	)
}
