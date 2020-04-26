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
	"bufio"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"testing"
)

func TestExampleConfiguration(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	wb.RunCodCmd("example-config", "--create")

	_, err := wb.UncheckedRunCodCmd("example-config", "--create")
	require.Error(t, err)

	shellPidInt := wb.LaunchFakeShell()
	shellPid := strconv.Itoa(shellPidInt)
	wb.RunCodCmd("init", shellPid, "bash")
	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")
	out := wb.RunCodCmd("api", "complete-words", shellPid, "--", "1", "binaries/cat.py", "-")

	scan := bufio.NewScanner(strings.NewReader(out))
	var lines []string
	for scan.Scan() {
		lines = append(lines, scan.Text())
	}
	require.Nil(t, scan.Err())

	require.Equal(t, []string{
		"-A",
		"--show-all",
		"-b",
		"--number-nonblank",
		"-e",
		"-E",
		"--show-ends",
		"-n",
		"--number",
		"-s",
		"--squeeze-blank",
		"-t",
		"-T",
		"--show-tabs",
		"-u",
		"-v",
		"--show-nonprinting",
		"--help",
		"--version",
	}, lines)
}
