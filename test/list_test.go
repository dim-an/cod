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

	out := wb.RunCodCmd("list", "--format", "plain")

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

	out := wb.RunCodCmd("list", "--format", "plain")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/cat.py --help",
			"binaries/naval-fate.py --help",
		},
		commands,
	)

	wb.RunCodCmd("remove", "1")
	out = wb.RunCodCmd("list", "--format", "plain")
	commands = wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/naval-fate.py --help",
		},
		commands,
	)

	wb.RunCodCmd("remove", "2")
	out = wb.RunCodCmd("list", "--format", "plain")
	require.Equal(t, "", out)
}

func TestLearnListBasenameSelector(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	wb.RunCodCmd("learn", "--", "binaries/naval-fate.py", "--help")

	out := wb.RunCodCmd("list", "--format", "plain", "cat.py")

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

	out := wb.RunCodCmd("list", "--format", "plain", abs)

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

	out := wb.RunCodCmd("list", "--format", "plain", "2")

	commands := wb.ParseCodListCommands(out)

	require.Equal(t,
		[]string{
			"binaries/naval-fate.py --help",
		},
		commands,
	)
}

func TestLearnListRichDefault(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()
	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")
	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")

	out := wb.RunCodCmd("list")

	require.Contains(t, out, "ID")
	require.Contains(t, out, "Command")
	require.Contains(t, out, "Description")
	require.Contains(t, out, "Completions")
	require.Contains(t, out, "binaries/cat.py --help")
	require.Contains(t, out, "Concatenate FILE(s) to standard output.")
	require.Contains(t, out, "19")
}

func TestShowCommandDetails(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()
	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")
	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")

	out := wb.RunCodCmd("show", "cat.py")

	require.Contains(t, out, "ID: 1")
	require.Contains(t, out, "binaries/cat.py --help")
	require.Contains(t, out, "Description: Concatenate FILE(s) to standard output.")
	require.Contains(t, out, "Completions: 19")
	require.Contains(t, out, "--number")
	require.Contains(t, out, "number all output lines")
}
