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
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestDaemonSmoke(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	cmd := wb.NewCodCmd("daemon", "--foreground")
	err := cmd.Start()
	require.Nil(t, err)
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
}

func TestAttachDetach(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := wb.LaunchFakeShell()

	wb.RunCodCmd("init", strconv.Itoa(shellPid), "bash")
	daemonPid := wb.GetDaemonPid()
	checkProcessExists := func(pid int) bool {
		err := unix.Kill(pid, 0)
		var errno unix.Errno
		if errors.As(err, &errno) && errno == unix.ESRCH {
			return false
		}
		require.NoError(t, err)
		return true
	}
	require.True(t, checkProcessExists(daemonPid))

	wb.KillFakeShell(shellPid)

	daemonExists := true
	for i := 0; i < 200; i += 1 {
		daemonExists = checkProcessExists(daemonPid)
		if !daemonExists {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	if daemonExists {
		t.Fatal("timeout while waiting for daemon to exit")
	}
}
