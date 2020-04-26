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
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"path"
)

var AppName = "cod"
var GitSha = "<unknown-git-sha>"
var Version = fmt.Sprintf("0.0.1rc (compiled from: %v)", GitSha)

func fatal(err error) {
	_, err = fmt.Fprintf(os.Stderr, "%v: error: %v\n", AppName, err)
	if err != nil {
		panic(err)
	}
	os.Exit(1)
}

func verifyFatal(err error) {
	if err != nil {
		fatal(err)
	}
}

func getAppName() string {
	exe, err := os.Executable()
	verifyFatal(err)
	return path.Base(exe)
}

func main() {
	AppName = getAppName()

	//
	// Arguments
	//

	var shell string
	var pid uint
	var foreground bool
	var selectors []string
	var createConfig bool

	addShellArg := func(c *kingpin.CmdClause) *kingpin.CmdClause {
		c.Arg("shell", "Shell name (bash, zsh or fish).").Required().StringVar(&shell)
		return c
	}

	addPidArg := func(c *kingpin.CmdClause) *kingpin.CmdClause {
		c.Arg("pid", "PID of the shell").Required().UintVar(&pid)
		return c
	}

	app := kingpin.New("cod", "Shell autocomplete generator based on `--help' texts.")
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Version(Version)

	learn := app.Command("learn", "Learn new completions from help command.")
	learnArgs := learn.Arg("subject", "Subject to learn.").Required().Strings()

	list := app.Command("list", "List known commands.").Alias("ls")
	list.Arg("selector", "Items to list.").StringsVar(&selectors)

	remove := app.Command("remove", "Forget known command").Alias("rm")
	remove.Arg("selector", "Items to remove.").Required().StringsVar(&selectors)

	update := app.Command("update", "Update known command")
	update.Arg("selector", "Items to update.").Required().StringsVar(&selectors)

	init := app.Command("init", "Output shell initialization script.")
	addPidArg(init)
	addShellArg(init)

	exampleConfig := app.Command("example-config", "print example configuration to stdout")
	exampleConfig.Flag(
		"create",
		"write configuration to config file instead of printing it to stdout (doesn't work if config file already exists)",
	).BoolVar(&createConfig)

	daemon := app.Command("daemon", "Start cod daemon.")
	daemon.Flag("foreground", "Run daemon in foreground.").BoolVar(&foreground)

	api := app.Command("api", "shell <-> cod interaction.").Hidden()

	apiAttach := api.Command("attach", "Attach daemon.").Hidden()
	addPidArg(apiAttach)
	addShellArg(apiAttach)

	apiBashCleanCompletions := api.Command("bash-clean-completions", "Clean completions").Hidden()
	appBase := apiBashCleanCompletions.Arg("executable", "executable to clean").Required().String()

	apiPollUpdates := api.Command("poll-updates", "poll updates from server").Hidden()
	addPidArg(apiPollUpdates)

	apiPostexec := api.Command("postexec", "check if command is help and suggest to learn it").Hidden()
	addPidArg(apiPostexec)
	apiPostexecCommand := apiPostexec.Arg("command", "command to analyze").Required().String()

	apiListClients := api.Command("list-clients", "help list all attached shells").Hidden()

	apiCompleteWords := api.Command("complete-words", "Get completions for given command line.").Hidden()
	addPidArg(apiCompleteWords)
	apiCompleteWordsCWord := apiCompleteWords.Arg("c-word", "Index of a word being completed.").Required().Int()
	apiCompleteWordsWords := apiCompleteWords.Arg("words", "Command line being completed.").Required().Strings()

	apiForkedDaemon := api.Command("forked-daemon", "Helper method to run a daemon").Hidden()
	notifyPid := apiForkedDaemon.Arg("pid", "Pid to notify after command start").Required().Int()

	//
	// Work
	//

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// commands
	case learn.FullCommand():
		learnMain(*learnArgs)
	case list.FullCommand():
		listMain(selectors)
	case init.FullCommand():
		initMain(pid, shell)
	case daemon.FullCommand():
		daemonMain(foreground)
	case remove.FullCommand():
		removeMain(selectors)
	case update.FullCommand():
		updateMain(selectors)
	case exampleConfig.FullCommand():
		exampleConfigMain(createConfig)

	// api
	case apiAttach.FullCommand():
		shellApiAttachMain(pid, shell)
	case apiPollUpdates.FullCommand():
		apiPollUpdatesMain(pid)
	case apiPostexec.FullCommand():
		apiPostexecMain(pid, *apiPostexecCommand)
	case apiCompleteWords.FullCommand():
		apiCompleteWordsMain(pid, *apiCompleteWordsCWord, *apiCompleteWordsWords)
	case apiListClients.FullCommand():
		apiListClientsMain()
	case apiForkedDaemon.FullCommand():
		forkedDaemonMain(*notifyPid)
	case apiBashCleanCompletions.FullCommand():
		apiBashCleanCompletionsMain(*appBase)
	default:
		panic("unexpected command")
	}
}
