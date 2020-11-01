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
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateReplace(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	tmpCat := wb.InTmpDataPath("foo")
	wb.CopyFile("binaries/foo_v1.py", tmpCat)
	wb.RunCodCmd("learn", "--", tmpCat, "--help")
	out := wb.RunCodCmd("api", "complete-words", shellPid, "--", "1", tmpCat, "--")
	completions := wb.SplitLines(out)
	sort.Strings(completions)
	require.Equal(t, []string{"--bar1", "--foo1"}, completions)

	wb.CopyFile("binaries/foo_v2.py", tmpCat)
	wb.RunCodCmd("update", "**")

	out = wb.RunCodCmd("list")
	parsed := wb.ParseCodListCommands(out)
	require.Equal(t,
		[]string{
			"foo --help",
		},
		parsed)

	out = wb.RunCodCmd("api", "complete-words", shellPid, "--", "1", tmpCat, "--")
	completions = wb.SplitLines(out)
	sort.Strings(completions)
	require.Equal(t, []string{"--bar2", "--foo2", "--qux2"}, completions)
}

func TestUpdateMerge(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	tmpCat := wb.InTmpDataPath("foo")
	wb.CopyFile("binaries/foo_v1.py", tmpCat)
	wb.RunCodCmd("learn", "--", tmpCat, "--help")

	out := wb.RunCodCmd("list")
	parsed := wb.ParseCodListCommands(out)
	require.Equal(t, len(parsed), 1)

	wb.CopyFile("binaries/foo_v2.py", tmpCat)
	wb.RunCodCmd("learn", "--", tmpCat, "gg", "--help")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListCommands(out)
	require.Equal(
		t,
		[]string{
			"foo --help",
			"foo gg --help",
		}, parsed)

	wb.RunCodCmd("update", "**")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListCommands(out)
	require.Equal(t, 1, len(parsed))
}

func TestUpdateNoChange(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	out := wb.RunCodCmd("list")
	parsed := wb.ParseCodListMap(out)
	require.Equal(t, len(parsed), 1)

	wb.RunCodCmd("update", "**")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListMap(out)
	require.Equal(t, len(parsed), 1)
}

func TestUpdateBroken(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	tmpCat := wb.InTmpDataPath("cat1")
	wb.CopyFile("binaries/cat.py", tmpCat)
	wb.RunCodCmd("learn", "--", tmpCat, "--help")
	out := wb.RunCodCmd("list")
	parsed := wb.ParseCodListMap(out)
	require.Equal(t, len(parsed), 1)

	err := os.Remove(tmpCat)
	require.Nil(t, err)

	wb.RunCodCmd("update", "**")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListMap(out)
	require.Equal(t, make(map[int]string), parsed)
}
