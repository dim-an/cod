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
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseArgparse(t *testing.T) {
	parseCompletions := func(executablePath, text string) (res []string) {
		ctx, err := makeParseContext(executablePath, text)
		require.NoError(t, err)

		parseResult, err := makeArgparseParser().Parse(ctx)
		require.NoError(t, err)
		for idx := range parseResult.completions {
			res = append(res, parseResult.completions[idx].Flag)
		}
		return
	}

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
		parseCompletions("/usr/bin/asciinema", asciicinemaHelp),
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
		parseCompletions("/home/user/.local/bin/do.py", doPyHelp),
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
