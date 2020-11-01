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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dim-an/cod/datastore"
	"github.com/dim-an/cod/server"
	"github.com/dim-an/cod/shells"
	"github.com/dim-an/cod/util"
)

func summarizeLearning(rsp *server.AddHelpPageResponse) (err error) {
	var msg string
	if rsp.Status == datastore.AddHelpPageStatusNew {
		examples := ""

		for i := range rsp.HelpPage.Completions {
			if i > 0 {
				examples += " "
			}
			cur := fmt.Sprintf("%q", rsp.HelpPage.Completions[i].Flag)
			if i == 0 || len(examples+cur) < 35 {
				examples += cur
			} else {
				examples += fmt.Sprintf("and %v more", len(rsp.HelpPage.Completions)-i)
				break
			}
		}

		msg = fmt.Sprintf("cod: learned completions: %s\n", examples)
	} else {
		if rsp.Status != datastore.AddHelpPageStatusUpdated {
			panic(fmt.Errorf("unexpected status: %v", rsp.Status))
		}
		msg = fmt.Sprintf("cod: updated completions\n")
	}

	ui := NewUI()
	_, err = fmt.Print(ui.Styled("green", msg))
	return
}

func apiBashCleanCompletionsMain(appBase string) {
	lines, err := shells.BashRemoveCompletions(appBase, os.Stdin)
	verifyFatal(err)
	for _, l := range lines {
		_, err = os.Stdout.WriteString(l)
		util.VerifyPanic(err)

		_, err = os.Stdout.WriteString("\n")
		util.VerifyPanic(err)
	}
}

func apiPollUpdatesMain(pid uint) {
	app := NewApplication()
	defer app.Close()

	req := server.PollUpdatesRequest{
		Pid: int(pid),
	}
	rsp := server.PollUpdatesResponse{}

	err := app.Client().Request(&req, &rsp)
	verifyFatal(err)

	for _, line := range rsp.Script {
		fmt.Println(line)
	}
}

func apiPostexecMain(pid uint, command string) {
	app := NewApplication()

	dir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	req := server.ParseCommandLineRequest{
		Pid:         int(pid),
		CommandLine: command,
		Dir:         dir,
		Env:         os.Environ(),
	}
	rsp := server.ParseCommandLineResponse{}

	err = app.Client().Request(&req, &rsp)
	if err != nil {
		if server.GetErrorCode(err) == server.BinaryNotFound {
			return
		}
		fatal(err)
	}

	if len(rsp.Args) == 0 || !rsp.IsHelpCommand || rsp.PolicyMode == datastore.PolicyIgnore {
		return
	}

	if rsp.PolicyMode == datastore.PolicyTrust {
		// send update request
		req := server.AddHelpPageRequest{
			Command: datastore.Command{
				Args: rsp.Args,
				Env:  append(os.Environ(), rsp.Env...),
				Dir:  dir,
			},
		}
		rsp := server.AddHelpPageResponse{}

		err = app.Client().Request(&req, &rsp)
		verifyFatal(err)
		err = summarizeLearning(&rsp)
		verifyFatal(err)
		return
	}

	ui := NewUI()

	// https://en.wikipedia.org/wiki/Box_Drawing_(Unicode_block)
	shortOptions := fmt.Sprintf(
		"┌──> %v\n└─── cod: learn this command? [yn?] > ",
		strings.Join(rsp.Args, " "),
	)

	fullOptions := "\n"
	fullOptions += " y => Yes and enable autoupdates for this commands\n"
	fullOptions += " n => Not now\n"
	fullOptions += " ? => show this help\n"
	fullOptions += " \n"
	fullOptions += " You can setup rules in cod config file. Check:\n"
	fullOptions += "   $ cod help example-configuration\n"
	fullOptions += " \n"
	fullOptions += " > "

	fmt.Print(ui.Styled("green", shortOptions))

loop:
	for {
		r, err := ui.GetKeystroke("yn?")
		if err != nil {
			fatal(err)
		}

		switch r {
		case 'y':
			var policy datastore.Policy
			policy = datastore.PolicyTrust
			req := server.AddHelpPageRequest{
				Command: datastore.Command{
					Args: rsp.Args,
					Env:  append(os.Environ(), rsp.Env...),
					Dir:  dir,
				},
				Policy: policy,
			}
			rsp := server.AddHelpPageResponse{}

			err = app.Client().Request(&req, &rsp)
			verifyFatal(err)
			err = summarizeLearning(&rsp)
			verifyFatal(err)
			break loop
		case 'n':
			// do nothing
			break loop
		case '?':
			fmt.Print(ui.Styled("green", fullOptions))
			continue
		default:
			panic(fmt.Errorf("unknown keystroke %v", r))
		}
	}
}

func apiListClientsMain() {
	app := NewApplication()
	defer app.Close()

	req := server.ListClientsRequest{}
	rsp := server.ListClientsResponse{}

	err := app.Client().Request(&req, &rsp)
	verifyFatal(err)

	for _, client := range rsp.Clients {
		fmt.Printf("%v\t%v\n", client.Pid, client.Shell)
	}
}

func initMain(pid uint, shell string) {
	app := NewApplication()
	defer app.Close()

	logDir := app.Config().GetLogDir()
	err := util.CreateDirIfNotExists(logDir)
	verifyFatal(err)

	runDir := app.Config().GetRunDir()
	err = util.CreateDirIfNotExists(runDir)
	verifyFatal(err)

	err = daemonize(app.Config())
	verifyFatal(err)

	{ // attach
		rsp := server.AttachResponse{}
		req := server.AttachRequest{
			Pid:   int(pid),
			Shell: shell,
		}

		err = app.Client().Request(&req, &rsp)
		if err != nil {
			fatal(err)
		}
	}

	{ // init script
		req := server.InitScriptRequest{
			Pid: int(pid),
		}
		rsp := server.InitScriptResponse{}
		err = app.Client().Request(&req, &rsp)
		if err != nil {
			fatal(err)
		}

		for _, line := range rsp.Script {
			fmt.Println(line)
		}
	}
}

func apiCompleteWordsMain(_ uint, cword int, words []string) {
	app := NewApplication()
	defer app.Close()

	if len(words) == 0 {
		fatal(fmt.Errorf("command line cannot be empty"))
	}

	env := os.Environ()
	dir, err := os.Getwd()
	verifyFatal(err)
	executablePath, err := datastore.CanonizeExecutablePath(words[0], dir, util.GetPathVar(env), util.GetHomeVar(env))
	verifyFatal(err)

	req := server.CompleteWordsRequest{
		Words: append([]string{executablePath}, words[1:]...),
		CWord: cword,
	}
	rsp := server.CompleteWordsResponse{}
	err = app.Client().Request(&req, &rsp)
	verifyFatal(err)

	for _, c := range rsp.Completions {
		fmt.Println(c)
	}
}

func learnMain(helpCommand []string) {
	config, err := server.DefaultConfiguration()
	verifyFatal(err)

	dir, err := os.Getwd()
	verifyFatal(err)

	command := datastore.Command{
		Args: helpCommand,
		Env:  os.Environ(),
		Dir:  dir,
	}

	req := server.AddHelpPageRequest{
		Command: command,
	}

	client, err := server.NewClient(config)
	if err != nil {
		fatal(fmt.Errorf("cannot connect to daemon: %w", err))
	}

	var rsp server.AddHelpPageResponse
	err = client.Request(&req, &rsp)
	verifyFatal(err)
	err = summarizeLearning(&rsp)
	verifyFatal(err)
}

type byApplication []server.ListCommandsResponseItem

func (b byApplication) Len() int {
	return len(b)
}

func (b byApplication) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byApplication) Less(i, j int) bool {
	lhs := b[i]
	rhs := b[j]
	if lhs.Command == nil {
		return rhs.Command != nil
	} else if rhs.Command == nil {
		return false
	}
	if len(lhs.Command.Args) == 0 || len(rhs.Command.Args) == 0 {
		panic(fmt.Errorf("server returned empty command"))
	}

	lhsBase := filepath.Base(lhs.Command.Args[0])
	rhsBase := filepath.Base(rhs.Command.Args[0])
	switch strings.Compare(lhsBase, rhsBase) {
	case -1:
		return true
	case 1:
		return false
	}
	switch strings.Compare(lhs.Command.Args[0], rhs.Command.Args[0]) {
	case -1:
		return true
	case 1:
		return false
	}
	return lhs.Id < rhs.Id
}

func listMain(selectors []string) {
	app := NewApplication()
	defer app.Close()

	req := server.ListCommandsRequest{}
	if len(selectors) > 0 {
		req.Selectors = selectors
	} else {
		req.Selectors = []string{"/**"}
	}
	rsp := server.ListCommandsResponse{}
	err := app.Client().Request(&req, &rsp)
	verifyFatal(err)

	sort.Sort(byApplication(rsp.CommandItems))
	for _, item := range rsp.CommandItems {
		quoted := "<broken>"
		if item.Command != nil {
			quoted = shells.Quote(item.Command.Args)
		}

		fmt.Printf("%v\t%v\n", item.Id, quoted)
	}
}

func removeMain(selectors []string) {
	app := NewApplication()
	defer app.Close()

	var ids []int64
	{
		req := server.ListCommandsRequest{
			Selectors: selectors,
		}
		rsp := server.ListCommandsResponse{}
		err := app.Client().Request(&req, &rsp)
		verifyFatal(err)
		for idx := range rsp.CommandItems {
			ids = append(ids, rsp.CommandItems[idx].Id)
		}
	}

	{
		req := server.RemoveCommandsRequest{}
		req.HelpPageIds = ids
		rsp := server.RemoveCommandsResponse{}
		err := app.Client().Request(&req, &rsp)
		verifyFatal(err)
	}
}

func updateMain(selectors []string) {
	app := NewApplication()
	defer app.Close()

	req := server.ListCommandsRequest{
		Selectors: selectors,
	}
	rsp := server.ListCommandsResponse{}
	err := app.Client().Request(&req, &rsp)
	verifyFatal(err)

	sort.Sort(byApplication(rsp.CommandItems))

	for _, item := range rsp.CommandItems {
		if item.Command == nil {
			continue
		}
		req := server.UpdateHelpPageRequest{
			Id:      item.Id,
			Command: *item.Command,
		}
		rsp := server.UpdateHelpPageResponse{}
		err = app.Client().Request(&req, &rsp)
		verifyFatal(err)
	}
}

func exampleConfigMain(createConfig bool) {
	if !createConfig {
		_, err := os.Stdout.WriteString(ExampleConfiguration)
		verifyFatal(err)
		return
	}
	configuration, err := server.DefaultConfiguration()
	verifyFatal(err)

	configPath := configuration.GetUserConfiguration()
	configDir, _ := filepath.Split(configPath)

	stat, err := os.Stat(configDir)
	if err == nil {
		if !stat.IsDir() {
			fatal(fmt.Errorf("not a directory: %s", configDir))
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(configDir, 0755)
		verifyFatal(err)
	} else {
		fatal(err)
	}

	stat, err = os.Stat(configPath)
	if err == nil {
		fatal(fmt.Errorf("already exists: %s", configPath))
	} else if !errors.Is(err, os.ErrNotExist) {
		fatal(err)
	}

	f, err := os.Create(configPath)
	verifyFatal(err)
	defer func() {
		err = f.Close()
		verifyFatal(err)
	}()

	_, err = f.WriteString(ExampleConfiguration)
	verifyFatal(err)
}
