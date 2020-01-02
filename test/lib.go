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
	"cod/shells"
	"cod/util"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type Workbench struct {
	t                  *testing.T
	codBinary          string
	dirWithTests       string
	currentTestWorkDir string
	currentTestTmpDir  string
	systemTmpDir       string

	fakeShellProcMap map[int]*os.Process
}

func SetupWorkbench(t *testing.T) Workbench {
	systemTmpDir, err := ioutil.TempDir("", "cod-test-")
	util.VerifyPanic(err)

	workDir, err := os.Getwd()
	require.Nil(t, err)

	// Length of unix socket path is limited.
	// Sometimes test frameworks create very long paths.
	// Here we try to shorten it.
	shortWd := filepath.Join(systemTmpDir, "wd")
	err = os.Symlink(workDir, shortWd)
	require.Nil(t, err)

	testRoot := filepath.Join(shortWd, "test-data")

	err = util.CreateDirIfNotExists(testRoot)
	util.VerifyPanic(err)

	currentTestWorkDir := filepath.Join(testRoot, t.Name())

	err = util.Purge(currentTestWorkDir)
	util.VerifyPanic(err)
	err = os.Mkdir(currentTestWorkDir, 0755)
	util.VerifyPanic(err)

	testDataDir := filepath.Join(currentTestWorkDir, "tmp")
	err = os.Mkdir(testDataDir, 0755)
	util.VerifyPanic(err)

	codBinary := os.Getenv("COD_TEST_BINARY")
	if len(codBinary) == 0 {
		panic(fmt.Errorf("COD_TEST_BINARY is not specified"))
	}

	return Workbench{
		t:                  t,
		codBinary:          codBinary,
		dirWithTests:       workDir,
		currentTestWorkDir: currentTestWorkDir,
		currentTestTmpDir:  testDataDir,
		systemTmpDir:       systemTmpDir,

		fakeShellProcMap: make(map[int]*os.Process),
	}
}

func (wb *Workbench) CopyFile(src, dst string) {
	fileInfo, err := os.Stat(src)
	require.Nil(wb.t, err)

	in, err := os.Open(src)
	require.Nil(wb.t, err)
	defer func() {
		_ = in.Close()
	}()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
	require.Nil(wb.t, err)
	defer func() {
		err := out.Close()
		require.Nil(wb.t, err)
	}()

	_, err = io.Copy(out, in)
	require.Nil(wb.t, err)

	err = out.Sync()
	require.Nil(wb.t, err)
}

func (wb *Workbench) InTmpDataPath(path string) string {
	return filepath.Join(wb.currentTestTmpDir, path)
}

func (wb *Workbench) LaunchFakeShell() int {
	cmd := exec.Command("sleep", "10")
	err := cmd.Start()
	util.VerifyPanic(err)
	pid := cmd.Process.Pid
	wb.fakeShellProcMap[pid] = cmd.Process
	return pid
}

func (wb *Workbench) KillFakeShell(pid int) {
	log.Printf("killing shell: %v", pid)
	proc, ok := wb.fakeShellProcMap[pid]
	if !ok {
		panic(fmt.Errorf("trying to kill unknown pid: %v", pid))
	}
	err := proc.Kill()
	require.NoError(wb.t, err)
	_, err = proc.Wait()
	require.NoError(wb.t, err)
}

func (wb *Workbench) UncheckedRunCodCmd(args ...string) (string, error) {
	cmd := wb.NewCodCmd(args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (wb *Workbench) RunCodCmd(args ...string) string {
	return wb.RunCodCmdModifiedEnv(nil, args...)
}

func (wb *Workbench) RunCodCmdModifiedEnv(env map[string]string, args ...string) string {
	if env != nil {
		oldMap := make(map[string]string)
		for k, v := range env {
			oldMap[k] = os.Getenv(k)
			err := os.Setenv(k, v)
			require.NoError(wb.t, err)
		}
		defer func() {
			for k, v := range oldMap {
				err := os.Setenv(k, v)
				require.NoError(wb.t, err)
			}
		}()
	}
	output, err := wb.UncheckedRunCodCmd(args...)
	if err != nil {
		cmdString := shells.Quote(append([]string{"cod"}, args...))
		err = fmt.Errorf("%q error: %w; output: %q", cmdString, err, output)
	}
	require.NoError(wb.t, err)
	return output
}

func (wb *Workbench) NewCodCmd(args ...string) exec.Cmd {
	cmd := exec.Cmd{}
	cmd.Path = wb.codBinary
	cmd.Args = append([]string{"cod-test"}, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("XDG_CONFIG_HOME=%v", wb.getConfigHome()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("XDG_DATA_HOME=%v", wb.getDataHome()))

	return cmd
}

func (wb *Workbench) getConfigHome() string {
	return filepath.Join(wb.currentTestWorkDir, "config")
}

func (wb *Workbench) getDataHome() string {
	return filepath.Join(wb.currentTestWorkDir, "data")
}

func (wb *Workbench) Close() {
	if wb.t.Failed() {
		log.Printf("Printing logs of failed test")
		logDir := filepath.Join(wb.getDataHome(), "cod", "log")
		logFiles, err := ioutil.ReadDir(logDir)
		require.NoError(wb.t, err)

		printFile := func(path string) {
			logF, err := os.Open(path)
			if err != nil {
				log.Printf("cannot open file %s: %s", path, err)
				return
			}
			defer func() {
				_ = logF.Close()
			}()

			data, err := ioutil.ReadAll(logF)
			if err != nil {
				log.Printf("cannot read file %s: %s", path, err)
			}
			log.Printf(" content of %s\n<<<BEGIN>>>\n%s<<<END>>>", path, string(data))
		}

		for _, fileInfo := range logFiles {
			if fileInfo.IsDir() {
				continue
			}
			path := filepath.Join(logDir, fileInfo.Name())
			printFile(path)
		}
	} else {
		err := os.RemoveAll(wb.currentTestWorkDir)
		util.VerifyPanic(err)
	}

	for pid, proc := range wb.fakeShellProcMap {
		_ = proc.Kill()
		delete(wb.fakeShellProcMap, pid)
	}

	err := os.RemoveAll(wb.systemTmpDir)
	util.VerifyPanic(err)
}

func (wb *Workbench) GetDaemonPid() int {
	lockFile := filepath.Join(wb.getDataHome(), "cod", "var", "cod.lock")
	f, err := os.Open(lockFile)
	require.NoError(wb.t, err)
	text, err := ioutil.ReadAll(f)
	require.NoError(wb.t, err)
	pid, err := strconv.Atoi(string(text))
	require.NoError(wb.t, err)
	return pid
}

func (wb *Workbench) SplitLines(s string) []string {
	var res []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	require.Nil(wb.t, scanner.Err())
	return res
}

func (wb *Workbench) ParseCodListMap(s string) map[int]string {
	res := make(map[int]string)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		splitted := strings.SplitN(line, "\t", 2)
		if len(splitted) != 2 {
			wb.t.Errorf("unexpected line in output of list command: %q", line)
		}
		id, err := strconv.Atoi(splitted[0])
		require.Nil(wb.t, err)

		commandLine := strings.TrimPrefix(splitted[1], wb.dirWithTests+"/")
		commandLine = strings.TrimPrefix(commandLine, wb.currentTestTmpDir+"/")
		res[id] = commandLine

	}
	require.Nil(wb.t, scanner.Err())
	return res
}

func (wb *Workbench) ParseCodListCommands(s string) []string {
	m := wb.ParseCodListMap(s)

	var res []string
	for _, v := range m {
		res = append(res, v)
	}
	sort.Strings(res)
	return res
}
