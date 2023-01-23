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

package parse_doc

import (
	"sort"
	"testing"

	"github.com/dim-an/cod/datastore"
	"github.com/stretchr/testify/require"
)

func parseArgparseCompletions(t *testing.T, args []string, text string) (res []string) {
	ctx, err := makeParseContext(args, text)
	require.NoError(t, err)

	parseResult, err := makeArgparseParser().Parse(ctx)
	require.NoError(t, err)
	for idx := range parseResult.completions {
		res = append(res, parseResult.completions[idx].Flag)
	}
	return
}

func TestParseArgparse(t *testing.T) {
	require.Equal(
		t,
		[]string{
			"rec",
			"play",
			"cat",
			"upload",
			"auth",
			"-h",
			"--help",
			"--version",
		},
		parseArgparseCompletions(t, []string{"/usr/bin/asciinema", "--help"}, asciicinemaHelp),
	)

	require.Equal(
		t,
		[]string{
			"up",
			"continue",
			"abort",
			"complete",
			"-h",
			"--help",
			"-q",
			"--quiet",
			"-v",
			"--verbose",
		},
		parseArgparseCompletions(t, []string{"/home/user/.local/bin/do.py", "--help"}, doPyHelp),
	)

	require.Equal(
		t,
		[]string{
			"-h",
			"--help",
			"--destination",
			"--compute",
		},
		parseArgparseCompletions(t, []string{"/usr/bin/qu", "--help"}, quWriteFileHelp),
	)
}

func TestParseArgparsePython310(t *testing.T) {
	// In Python 3.10 we have newer format
	require.Equal(
		t,
		[]string{
			"sub-command1",
			"sub-command2",
			"-h",
			"--help",
			"--parser-argument",
		},
		parseArgparseCompletions(t, []string{"/usr/bin/argparse-python-3-10", "--help"}, argparsePython310Help),
	)
}

func TestParseArgparseContext(t *testing.T) {
	ctx, err := makeParseContext([]string{"/usr/bin/asciinema", "rec", "--help"}, asciinemaRecHelp)
	require.NoError(t, err)

	parseResult, err := makeArgparseParser().Parse(ctx)
	require.NoError(t, err)
	sort.Slice(parseResult.completions, func(i, j int) bool {
		return parseResult.completions[i].Flag < parseResult.completions[j].Flag
	})
	argparseContext := func(subCommand []string) datastore.FlagContext {
		return datastore.FlagContext{
			SubCommand: subCommand,
			Framework:  "argparse",
		}
	}
	require.Equal(
		t,
		[]datastore.Completion{
			{Flag: "--append", Context: argparseContext([]string{"rec"})},
			{Flag: "--command", Context: argparseContext([]string{"rec"})},
			{Flag: "--env", Context: argparseContext([]string{"rec"})},
			{Flag: "--help", Context: argparseContext([]string{"rec"})},
			{Flag: "--idle-time-limit", Context: argparseContext([]string{"rec"})},
			{Flag: "--overwrite", Context: argparseContext([]string{"rec"})},
			{Flag: "--quiet", Context: argparseContext([]string{"rec"})},
			{Flag: "--raw", Context: argparseContext([]string{"rec"})},
			{Flag: "--stdin", Context: argparseContext([]string{"rec"})},
			{Flag: "--title", Context: argparseContext([]string{"rec"})},
			{Flag: "--yes", Context: argparseContext([]string{"rec"})},
			{Flag: "-c", Context: argparseContext([]string{"rec"})},
			{Flag: "-e", Context: argparseContext([]string{"rec"})},
			{Flag: "-h", Context: argparseContext([]string{"rec"})},
			{Flag: "-i", Context: argparseContext([]string{"rec"})},
			{Flag: "-q", Context: argparseContext([]string{"rec"})},
			{Flag: "-t", Context: argparseContext([]string{"rec"})},
			{Flag: "-y", Context: argparseContext([]string{"rec"})},
			// FIXME: this is bug, we should only parse `-y` single time
			{Flag: "-y", Context: argparseContext([]string{"rec"})},
		},
		parseResult.completions,
	)
}

var asciicinemaHelp = `usage: asciinema [-h] [--version] {rec,play,cat,upload,auth} ...

Record and share your terminal sessions, the right way.

positional arguments:
  {rec,play,cat,upload,auth}
    rec                 Record terminal session
    play                Replay terminal session
    cat                 Print full output of terminal session
    upload              Upload locally saved terminal session to asciinema.org
    auth                Manage recordings on asciinema.org account

optional arguments:
  -h, --help            show this help message and exit
  --version             show program's version number and exit

example usage:
  Record terminal and upload it to asciinema.org:
    asciinema rec
  Record terminal to local file:
    asciinema rec demo.cast
  Record terminal and upload it to asciinema.org, specifying title:
    asciinema rec -t "My git tutorial"
  Record terminal to local file, limiting idle time to max 2.5 sec:
    asciinema rec -i 2.5 demo.cast
  Replay terminal recording from local file:
    asciinema play demo.cast
  Replay terminal recording hosted on asciinema.org:
    asciinema play https://asciinema.org/a/difqlgx86ym6emrmd8u62yqu8
  Print full output of recorded session:
    asciinema cat demo.cast

For help on a specific command run:
  asciinema <command> -h
`

var asciinemaRecHelp = `usage: asciinema rec [-h] [--stdin] [--append] [--raw] [--overwrite]
                     [-c COMMAND] [-e ENV] [-t TITLE] [-i IDLE_TIME_LIMIT]
                     [-y] [-q]
                     [filename]

positional arguments:
  filename              filename/path to save the recording to

optional arguments:
  -h, --help            show this help message and exit
  --stdin               enable stdin recording, disabled by default
  --append              append to existing recording
  --raw                 save only raw stdout output
  --overwrite           overwrite the file if it already exists
  -c COMMAND, --command COMMAND
                        command to record, defaults to $SHELL
  -e ENV, --env ENV     list of environment variables to capture, defaults to
                        SHELL,TERM
  -t TITLE, --title TITLE
                        title of the asciicast
  -i IDLE_TIME_LIMIT, --idle-time-limit IDLE_TIME_LIMIT
                        limit recorded idle time to given number of seconds
  -y, --yes             answer "yes" to all prompts (e.g. upload confirmation)
  -q, --quiet           be quiet, suppress all notices/warnings (implies -y)
`

var doPyHelp = `usage: do.py [-h] [-q | -v] command ...

Pretty useful program that does things.

positional arguments:
  command        command to run
    up           do update
    continue     continue updating
    abort        abort updating
    complete     complete updating: and do first thing then second thing then
                 third thing then fourth thing then fifth thing then sixth
                 thing then seventh thing

optional arguments:
  -h, --help     show this help message and exit
  -q, --quiet    minimize logging
  -v, --verbose  maximize logging
`

var argparsePython310Help = `usage: argparse-python-3-10 [-h] [--parser-argument PARSER_ARGUMENT] {sub-command1,sub-command2} ...

positional arguments:
  {sub-command1,sub-command2}
    sub-command1        some help
    sub-command2        some help

options:
  -h, --help            show this help message and exit
  --parser-argument PARSER_ARGUMENT
                        some help
`
