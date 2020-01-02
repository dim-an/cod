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

package server

import (
	"bufio"
	"bytes"
	"cod/datastore"
	"cod/parse_doc"
	"cod/shells"
	"cod/util"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server interface {
	Serve() error
	Close() error
}

func NewServer(cfg *Configuration) (server Server, err error) {
	serverImpl := &serverImpl{
		configuration: cfg,
		shellInfoMap:  make(map[int]*shellInfo),
	}
	go serverImpl.waitAttach()
	go serverImpl.trimLogs()

	err = serverImpl.listen()
	if err != nil {
		return
	}
	server = serverImpl
	return
}

type shellInfo struct {
	pid                 int
	shell               string
	scriptGenerator     shells.ShellScriptGenerator
	executablesToUpdate map[string]bool
}

type serverImpl struct {
	listener net.Listener

	wg sync.WaitGroup

	mutex         sync.Mutex
	initialized   bool
	configuration *Configuration
	shellInfoMap  map[int]*shellInfo
	storage       datastore.Storage

	userConfiguration UserConfiguration
}

func (s *serverImpl) Serve() (err error) {
	log.Printf("Start serving requests")
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			break
		}
		s.wg.Add(1)
		go s.handleConnectionProc(conn)
	}

	s.wg.Wait()
	return
}

func (s *serverImpl) Close() (err error) {
	return
}

func (s *serverImpl) waitAttach() {
	time.Sleep(time.Second * 3)
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.initialized {
		log.Fatal("daemon was not attached, exiting")
	}
}

func (s *serverImpl) listen() (err error) {
	socketFile := s.configuration.GetSocketFile()
	stat, err := os.Stat(socketFile)
	log.Printf("stat %s: %v %v", socketFile, stat, err)
	s.listener, err = net.Listen("unix", socketFile)
	if err != nil {
		err = fmt.Errorf("cannot listen socket %s: %w", socketFile, err)
	}
	return
}

func (s *serverImpl) handleConnectionProc(conn net.Conn) {
	defer s.wg.Done()
	closeConn := func() {
		err := conn.Close()
		if err != nil {
			log.Printf("Cannot close connection: %v", err)
		}
	}

	defer closeConn()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		reqData := scanner.Bytes()

		log.Printf("Received request: %v", string(reqData))
		rspData, err := s.handleRequest(reqData)
		if err != nil {
			rspData = MarshalResponse(nil, err, nil)
		}
		log.Printf("Sending response: %v", string(rspData))

		if !bytes.HasSuffix(rspData, []byte{byte('\n')}) {
			rspData = append(rspData, byte('\n'))
		}
		_, err = conn.Write(rspData)
		if err != nil {
			break
		}
	}
}

func (s *serverImpl) verifyInitialized() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.initialized {
		return fmt.Errorf("server is not initialized yet")
	}
	return nil
}

func (s *serverImpl) handleRequest(reqData []byte) (rspData []byte, err error) {
	name, payload, err := UnmarshalRequest(reqData)
	warner := &util.Warner{}
	if err != nil {
		return
	} else {
		if name != "AttachRequest" {
			err = s.verifyInitialized()
			if err != nil {
				return
			}
		}

		switch name {
		case "DetachRequest":
			req := DetachRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleDetach(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "AttachRequest":
			req := AttachRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleAttach(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "BashCompletionRequest":
			req := BashCompletionRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleBashCompletion(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "InitScriptRequest":
			req := InitScriptRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleInitScript(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "ListClientsRequest":
			req := ListClientsRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleListClients(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "ListCommandsRequest":
			req := ListCommandsRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleListCommands(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "RemoveCommandsRequest":
			req := RemoveCommandsRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleRemoveCommands(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "AddHelpPageRequest":
			req := AddHelpPageRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleAddHelpPage(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		case "PollUpdatesRequest":
			req := PollUpdatesRequest{}
			CastRequestPayload(payload, &req)
			rsp, err, warns := s.handlePollUpdates(&req)
			rspData = MarshalResponse(&rsp, err, warns)
		case "ParseCommandLineRequest":
			req := ParseCommandLineRequest{}
			CastRequestPayload(payload, &req)
			rsp, err, warns := s.handleParseCommandLine(&req)
			rspData = MarshalResponse(&rsp, err, warns)
		case "UpdateHelpPageRequest":
			req := UpdateHelpPageRequest{}
			CastRequestPayload(payload, &req)
			rsp, err := s.handleUpdateHelpPageRequest(&req, warner)
			rspData = MarshalResponse(&rsp, err, warner.Warns)
		default:
			err = fmt.Errorf("unknown request: %v", name)
			return
		}
	}
	return
}

func (s *serverImpl) initializeStorage() (err error) {
	s.storage, err = datastore.NewSqliteStorage(s.configuration.GetCompletionsSqliteDb())
	if err != nil {
		return
	}
	s.userConfiguration, err = LoadUserConfiguration(
		s.configuration.GetUserConfiguration(),
		s.configuration.GetHomeDir(),
	)
	if err != nil {
		return
	}
	return
}

func (s *serverImpl) getWatchedPids() []int {
	var res []int
	for p := range s.shellInfoMap {
		res = append(res, p)
	}
	sort.Ints(res)
	return res
}

func (s *serverImpl) notifyExecutableUpdate(executablePath string) {
	for _, si := range s.shellInfoMap {
		si.executablesToUpdate[executablePath] = true
	}
}

func (s *serverImpl) handleAttach(req *AttachRequest, _ *util.Warner) (rsp AttachResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.initialized {
		err = s.initializeStorage()
		if err != nil {
			return
		}
		s.initialized = true
	}

	scriptGenerator, err := shells.NewShellScriptGenerator(req.Shell)
	if err != nil {
		return
	}

	s.shellInfoMap[req.Pid] = &shellInfo{
		pid:                 req.Pid,
		shell:               req.Shell,
		scriptGenerator:     scriptGenerator,
		executablesToUpdate: make(map[string]bool),
	}
	log.Printf("Watched pids: %v", s.getWatchedPids())
	go s.waitPidProc(req.Pid)
	s.initialized = true
	return
}

func (s *serverImpl) handleBashCompletion(req *BashCompletionRequest, _ *util.Warner) (rsp BashCompletionResponse, err error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	completions, err := s.storage.GetCompletions(req.ExecutablePath)
	if err != nil {
		return
	}

	for _, completion := range completions {
		if strings.HasPrefix(completion.Flag, req.Word) {
			rsp.Completions = append(rsp.Completions, completion.Flag)
		}
	}

	return
}

func (s *serverImpl) handleDetach(req *DetachRequest, _ *util.Warner) (rsp DetachResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.shellInfoMap, req.Pid)
	log.Printf("Watched pids: %v", s.getWatchedPids())
	if len(s.shellInfoMap) == 0 {
		err = s.listener.Close()
	}
	return
}

func (s *serverImpl) handleInitScript(req *InitScriptRequest, _ *util.Warner) (rsp InitScriptResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info, err := s.getShellInfo(req.Pid)
	if err != nil {
		return
	}

	rsp.Script = info.scriptGenerator.GetPreamble()
	helpPageList, err := s.storage.GetAllCompletions()
	if err != nil {
		return
	}
	for _, helpPage := range helpPageList {
		rsp.Script = append(
			rsp.Script, info.scriptGenerator.GenerateCompletions(helpPage.ExecutablePath, helpPage.Completions)...,
		)
	}
	return
}

func (s *serverImpl) handleListClients(_ *ListClientsRequest, _ *util.Warner) (rsp ListClientsResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, shellInfo := range s.shellInfoMap {
		rsp.Clients = append(rsp.Clients, ShellAndPid{
			Shell: shellInfo.shell,
			Pid:   shellInfo.pid,
		})
	}
	return
}

var numberRe = regexp.MustCompile("^\\d+$")

func (s *serverImpl) handleListCommands(req *ListCommandsRequest, _ *util.Warner) (rsp ListCommandsResponse, err error) {
	idFilter := make(map[int64]bool)
	var globs []util.Selector
	for _, selector := range req.Selectors {
		switch {
		case numberRe.MatchString(selector):
			var pid int64
			pid, err = strconv.ParseInt(selector, 10, 64)
			if err != nil {
				return
			}
			idFilter[pid] = true
		default:
			var g util.Selector
			g, err = util.CompileSelector(selector, s.configuration.GetHomeDir())
			if err != nil {
				return
			}
			globs = append(globs, g)
		}
	}

	commands, err := s.storage.ListCommands()
	if err != nil {
		return
	}

	for id, command := range commands {
		take := false
		if _, ok := idFilter[id]; ok {
			take = true
		} else if command != nil && len(command.Args) > 0 {
			for _, g := range globs {
				if g.MatchString(command.Args[0]) {
					take = true
					break
				}
			}
		}

		if take {
			item := ListCommandsResponseItem{
				Id:      id,
				Command: command,
			}
			rsp.CommandItems = append(rsp.CommandItems, item)
		}
	}
	return
}

func (s *serverImpl) handleRemoveCommands(req *RemoveCommandsRequest, _ *util.Warner) (rsp RemoveCommandsResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, id := range req.HelpPageIds {
		var executablePath string
		executablePath, err = s.storage.RemoveHelpPage(id)
		if err != nil {
			return
		}
		s.notifyExecutableUpdate(executablePath)
	}
	return
}

func (s *serverImpl) runHelpCommand(command datastore.Command, ctx context.Context) (helpPage *datastore.HelpPage, err error) {
	if len(command.Args) == 0 {
		err = fmt.Errorf("command is empty")
		return
	}

	executablePath, err := datastore.CanonizeExecutablePath(
		command.Args[0],
		command.Dir,
		util.GetPathVar(command.Env),
		util.GetHomeVar(command.Env),
	)

	if err != nil {
		return
	}

	cmd := exec.CommandContext(ctx, executablePath)
	cmd.Args = command.Args
	cmd.Args[0] = executablePath
	cmd.Env = command.Env
	cmd.Dir = command.Dir
	cmd.Stdin = nil

	helpBytes, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%w: %v", err, string(helpBytes))
		return
	}
	helpPage, err = parse_doc.ParseHelp(executablePath, string(helpBytes))
	if err != nil {
		return
	}
	helpPage.Command = command

	return
}

func (s *serverImpl) handleAddHelpPage(req *AddHelpPageRequest, _ *util.Warner) (rsp AddHelpPageResponse, err error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*1)
	helpPage, err := s.runHelpCommand(req.Command, ctx)
	if err != nil {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var status datastore.AddHelpPageStatus
	status, err = s.storage.AddHelpPage(helpPage, req.Policy)
	if err != nil {
		return
	}

	s.notifyExecutableUpdate(helpPage.ExecutablePath)

	rsp.HelpPage = *helpPage
	rsp.Status = status
	return
}

func (s *serverImpl) handlePollUpdates(req *PollUpdatesRequest) (rsp PollUpdatesResponse, err error, warns []util.Warning) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info, err := s.getShellInfo(req.Pid)
	if err != nil {
		return
	}

	for executablePath := range info.executablesToUpdate {
		var completions []datastore.Completion
		completions, err = s.storage.GetCompletions(executablePath)
		if err != nil {
			return
		}

		rsp.Script = append(rsp.Script, info.scriptGenerator.ResetCommand(executablePath)...)
		if len(completions) > 0 {
			rsp.Script = append(rsp.Script, info.scriptGenerator.GenerateCompletions(executablePath, completions)...)
		}
		delete(info.executablesToUpdate, executablePath)
	}
	return
}

func (s *serverImpl) handleParseCommandLine(req *ParseCommandLineRequest) (rsp ParseCommandLineResponse, err error, warns []util.Warning) {
	rsp.Env, rsp.Args, err = shells.ParseSimpleCommand(req.CommandLine)
	if err != nil {
		if errors.Is(err, shells.ErrCommandNotSimple) {
			err = nil
		}
		return
	}

	rsp.IsHelpCommand = false
loop:
	for _, a := range rsp.Args {
		switch a {
		case "--help":
			rsp.IsHelpCommand = true
			break loop
		case "--":
			break loop
		}
	}

	var executablePath string
	if len(rsp.Args) > 0 {
		executablePath, err = datastore.CanonizeExecutablePath(
			rsp.Args[0],
			req.Dir,
			util.GetPathVar(req.Env),
			util.GetHomeVar(req.Env),
		)
		if err != nil {
			return
		}
		rsp.Args[0] = executablePath
		var policy datastore.Policy
		policy, err = s.storage.GetCommandPolicy(rsp.Args)
		if err != nil {
			return
		}
		if policy != datastore.PolicyUnknown {
			rsp.PolicyMode = policy
		} else if policy = s.userConfiguration.GetExecutablePolicy(rsp.Args[0]); policy != datastore.PolicyUnknown {
			rsp.PolicyMode = policy
		} else {
			rsp.PolicyMode = datastore.PolicyAsk
		}
	}

	return
}

func (s *serverImpl) handleUpdateHelpPageRequest(req *UpdateHelpPageRequest, warner *util.Warner) (rsp UpdateHelpPageResponse, err error) {
	cmd := req.Command
	_, err = os.Stat(cmd.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			runDir := s.configuration.GetRunDir()
			warner.Warnf(
				"updating %v ; will use %q instead non existing %q",
				shells.Quote(cmd.Args),
				runDir,
				cmd.Dir,
			)
			cmd.Dir = s.configuration.GetRunDir()
		} else {
			return
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*1)
	helpPage, err := s.runHelpCommand(cmd, ctx)
	if err != nil {
		warner.Warnf("error running %v: %v", shells.Quote(cmd.Args), err)
		_, err = s.storage.RemoveHelpPage(req.Id)
	} else {
		_, err = s.storage.AddHelpPage(helpPage, datastore.PolicyUnknown)
	}

	return
}

func (s *serverImpl) getShellInfo(pid int) (info *shellInfo, err error) {
	info, ok := s.shellInfoMap[pid]
	if !ok {
		err = fmt.Errorf("unknown pid: %v", pid)
	}
	return
}

func (s *serverImpl) waitPidProc(pid int) {
	for {
		err := unix.Kill(pid, 0)
		if err != nil {
			log.Printf("Done waiting process %v: %v", pid, err)
			_, err := s.handleDetach(&DetachRequest{pid}, nil)
			util.VerifyPanic(err)
			break
		}
		time.Sleep(time.Millisecond * 300)
	}
}

var logPattern = regexp.MustCompile(`^cod[.]\d\d\d\d-\d\d-\d\d[.]log$`)

func (s *serverImpl) trimLogs() {
	maxLogCount := 7

	logDir := s.configuration.GetLogDir()
	fileInfoList, err := ioutil.ReadDir(logDir)
	if err != nil {
		log.Printf("cannot read log directory: %v", err)
		return
	}

	var logFileList []string
	for _, fileInfo := range fileInfoList {
		if fileInfo.IsDir() {
			continue
		}
		if logPattern.MatchString(fileInfo.Name()) {
			logFileList = append(logFileList, fileInfo.Name())
		}
	}

	if len(logFileList) <= maxLogCount {
		return
	}

	sort.Strings(logFileList)

	for _, name := range logFileList[:len(logFileList)-maxLogCount] {
		fullPath := filepath.Join(logDir, name)
		err = os.Remove(fullPath)
		if err != nil {
			log.Printf("cannot remove old log file %q: %v", fullPath, err)
		}
	}
}
