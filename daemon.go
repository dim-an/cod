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

package main

import (
	"errors"
	"fmt"
	"github.com/dim-an/cod/server"
	"github.com/dim-an/cod/shells"
	"github.com/dim-an/cod/util"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"time"
)

// https://stackoverflow.com/questions/17954432/creating-a-daemon-in-linux/17955149#17955149
func daemonMain(foreground bool) {
	configuration, err := server.DefaultConfiguration()
	verifyFatal(err)
	if foreground {
		daemonProc(configuration, 0)
	}

	logDir := configuration.GetLogDir()
	err = util.CreateDirIfNotExists(logDir)
	verifyFatal(err)

	runDir := configuration.GetRunDir()
	err = util.CreateDirIfNotExists(runDir)
	verifyFatal(err)

	executable, err := os.Executable()
	verifyFatal(err)

	sigqueue := make(chan os.Signal, 1)
	signal.Notify(sigqueue, unix.SIGUSR1)
	go func() {
		time.Sleep(time.Second * 2)
		sigqueue <- unix.SIGUSR2
	}()

	pid := os.Getpid()
	cmdArgs := []string{executable, "api", "forked-daemon", strconv.Itoa(pid)}
	cmd := exec.Command(executable, cmdArgs[1:]...)
	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("command failed %q: %w", shells.Quote(cmdArgs), err)
		fatal(err)
	}

	s := <-sigqueue
	if s != unix.SIGUSR1 {
		panic(fmt.Errorf("could not wait for daemon"))
	}
}

func cleanFile(file string) (err error) {
	err = os.Remove(file)
	if os.IsNotExist(err) {
		err = nil
	}
	return
}

func daemonProc(configuration server.Configuration, pidToNotify int) {
	err := os.Chdir("/")
	verifyFatal(err)

	log.SetPrefix(fmt.Sprintf("%v ", os.Getpid()))
	log.Printf("Starting daemon. Version: %q", Version)

	lockFileName := configuration.GetLockFile()
	lockFileFd, err := unix.Open(lockFileName, os.O_CREATE|os.O_RDWR, 0600)
	verifyFatal(err)

	pidStr := []byte(strconv.Itoa(os.Getpid()))
	_, err = unix.Write(lockFileFd, pidStr)
	if err != nil {
		log.Panic(fmt.Errorf("cannot write to %s: %w", lockFileName, err))
	}

	log.Printf("Locking file: %s", lockFileName)
	err = unix.Flock(lockFileFd, unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		log.Panic(fmt.Errorf("cannot lock file %s: %w", lockFileName, err))
	}

	log.Printf("Removing old unix socket")
	socketFile := configuration.GetSocketFile()
	err = cleanFile(socketFile)
	if err != nil {
		log.Panic(fmt.Errorf("cannot remove %s: %w", socketFile, err))
	}

	log.Printf("Launching server")
	s, err := server.NewServer(&configuration)
	verifyFatal(err)
	defer func() {
		err = s.Close()
		verifyFatal(err)
	}()

	if pidToNotify != 0 {
		err = unix.Kill(pidToNotify, unix.SIGUSR1)
	}

	err = s.Serve()
	verifyFatal(err)

	log.Printf("Daemon is exiting normally")
	os.Exit(0)
}

func forkedDaemonMain(pidToNotify int) {
	configuration, err := server.DefaultConfiguration()
	verifyFatal(err)

	_, err = unix.Setsid()
	verifyFatal(err)

	sigqueue := make(chan os.Signal, 1)
	go sighandler(sigqueue)
	signal.Notify(sigqueue, unix.SIGHUP)

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	verifyFatal(err)
	err = unix.Dup2(int(devNull.Fd()), 0)
	verifyFatal(err)
	err = unix.Dup2(int(devNull.Fd()), 1)
	verifyFatal(err)
	err = devNull.Close()
	verifyFatal(err)

	logFilePath := path.Join(configuration.GetLogDir(), getLogFileBaseName(time.Now()))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	verifyFatal(err)
	err = unix.Dup2(int(logFile.Fd()), 2)
	verifyFatal(err)
	err = logFile.Close()
	verifyFatal(err)

	daemonProc(configuration, pidToNotify)
}

func sighandler(queue chan os.Signal) {
	for s := range queue {
		log.Println("Got signal:", s)
	}
}

func daemonize(config *server.Configuration) (err error) {
	checkDaemonIsRunning := func() (isRunning bool, err error) {
		var fd int
		fd, err = unix.Open(config.GetLockFile(), os.O_RDONLY, 0600)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
			isRunning = false
			return
		} else if err != nil {
			return
		}

		defer func() {
			err = unix.Close(fd)
		}()
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if errors.Is(err, unix.EWOULDBLOCK) {
			err = nil
			isRunning = true
			return
		} else if err != nil {
			return
		}
		isRunning = false
		err = unix.Flock(fd, unix.LOCK_UN)
		return
	}

	var isRunning bool
	isRunning, err = checkDaemonIsRunning()
	if err != nil {
		return
	}
	if isRunning {
		return
	}

	var executable string
	executable, err = os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(executable, "daemon")
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("\"%s daemonize\" error: %w", executable, err)
	}
	return
}
