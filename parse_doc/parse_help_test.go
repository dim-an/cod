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
	"testing"

	"github.com/dim-an/cod/datastore"
	"github.com/stretchr/testify/require"
)

var catHelp = `Usage: cat [OPTION]... [FILE]...
Concatenate FILE(s) to standard output.

With no FILE, or when FILE is -, read standard input.

  -A, --show-all           equivalent to -vET
  -e                       equivalent to -vE
      --help     display this help and exit

Examples:
  cat f - g  Output f's contents, then standard input, then g's contents.
  cat        Copy standard input to standard output.

GNU coreutils online help: <https://www.gnu.org/software/coreutils/>
Full documentation <https://www.gnu.org/software/coreutils/cat>
or available locally via: info '(coreutils) cat invocation'
`

func TestParseCatHelp(t *testing.T) {
	desc, err := ParseHelp([]string{"cat", "--help"}, catHelp)
	require.Nil(t, err)

	expected := datastore.HelpPage{
		ExecutablePath: "cat",
		Completions: []datastore.Completion{
			{Flag: "-A"},
			{Flag: "--show-all"},
			{Flag: "-e"},
			{Flag: "--help"},
		},
		CheckSum: "4a8d01dde2483ad006b8f5ac2f599f9369287730",
	}
	require.Equal(t, expected, *desc)
}

var quWriteFileHelp = `
usage: qu write-file [-h] [--destination DESTINATION]
                     [--compute]
                     [destination]

positional arguments:
  destination           destination see also
                        http://example.com/

optional arguments:
  -h, --help            show this help message and exit
  --destination DESTINATION
                        destination see also http://example.com/
  --compute             compute file content
`

func TestParseQuWriteFileHelp(t *testing.T) {
	desc, err := ParseHelp([]string{"qu", "--help"}, quWriteFileHelp)
	require.Nil(t, err)

	expectedContext := datastore.FlagContext{
		SubCommand: []string{"write-file"},
		Framework:  "argparse",
	}
	expected := datastore.HelpPage{
		ExecutablePath: "qu",
		Completions: []datastore.Completion{
			{Flag: "-h", Context: expectedContext},
			{Flag: "--help", Context: expectedContext},
			{Flag: "--destination", Context: expectedContext},
			{Flag: "--compute", Context: expectedContext},
		},
		CheckSum: "2fee90df109afb526a5fa229861677497d8baf14",
	}
	require.Equal(t, expected, *desc)
}

var lsHelp = `
Usage: ls [OPTION]... [FILE]...
List information about the FILEs (the current directory by default).
Sort entries alphabetically if none of -cftuvSUX nor --sort is specified.

Mandatory arguments to long options are mandatory for short options too.
  -a, --all                  do not ignore entries starting with .
  -A, --almost-all           do not list implied . and ..
      --author               with -l, print the author of each file
  -b, --escape               print C-style escapes for nongraphic characters
      --block-size=SIZE      with -l, scale sizes by SIZE when printing them;
                               e.g., '--block-size=M'; see SIZE format below
  -B, --ignore-backups       do not list implied entries ending with ~
  -c                         with -lt: sort by, and show, ctime (time of last
                               modification of file status information);
                               with -l: show ctime and sort by name;
                               otherwise: sort by ctime, newest first
  -C                         list entries by columns
      --color[=WHEN]         colorize the output; WHEN can be 'always' (default
                               if omitted), 'auto', or 'never'; more info below
  -d, --directory            list directories themselves, not their contents
  -D, --dired                generate output designed for Emacs' dired mode
  -f                         do not sort, enable -aU, disable -ls --color
  -F, --classify             append indicator (one of */=>@|) to entries
      --file-type            likewise, except do not append '*'
      --format=WORD          across -x, commas -m, horizontal -x, long -l,
                               single-column -1, verbose -l, vertical -C
      --full-time            like -l --time-style=full-iso
  -g                         like -l, but do not list owner
      --group-directories-first
                             group directories before files;
                               can be augmented with a --sort option, but any
                               use of --sort=none (-U) disables grouping
  -G, --no-group             in a long listing, don't print group names
  -h, --human-readable       with -l and -s, print sizes like 1K 234M 2G etc.
      --si                   likewise, but use powers of 1000 not 1024
  -H, --dereference-command-line
                             follow symbolic links listed on the command line
      --dereference-command-line-symlink-to-dir
                             follow each command line symbolic link
                               that points to a directory
      --hide=PATTERN         do not list implied entries matching shell PATTERN
                               (overridden by -a or -A)
      --hyperlink[=WHEN]     hyperlink file names; WHEN can be 'always'
                               (default if omitted), 'auto', or 'never'
      --indicator-style=WORD  append indicator with style WORD to entry names:
                               none (default), slash (-p),
                               file-type (--file-type), classify (-F)
  -i, --inode                print the index number of each file
  -I, --ignore=PATTERN       do not list implied entries matching shell PATTERN
  -k, --kibibytes            default to 1024-byte blocks for disk usage;
                               used only with -s and per directory totals
  -l                         use a long listing format
  -L, --dereference          when showing file information for a symbolic
                               link, show information for the file the link
                               references rather than for the link itself
  -m                         fill width with a comma separated list of entries
  -n, --numeric-uid-gid      like -l, but list numeric user and group IDs
  -N, --literal              print entry names without quoting
  -o                         like -l, but do not list group information
  -p, --indicator-style=slash
                             append / indicator to directories
  -q, --hide-control-chars   print ? instead of nongraphic characters
      --show-control-chars   show nongraphic characters as-is (the default,
                               unless program is 'ls' and output is a terminal)
  -Q, --quote-name           enclose entry names in double quotes
      --quoting-style=WORD   use quoting style WORD for entry names:
                               literal, locale, shell, shell-always,
                               shell-escape, shell-escape-always, c, escape
                               (overrides QUOTING_STYLE environment variable)
  -r, --reverse              reverse order while sorting
  -R, --recursive            list subdirectories recursively
  -s, --size                 print the allocated size of each file, in blocks
  -S                         sort by file size, largest first
      --sort=WORD            sort by WORD instead of name: none (-U), size (-S),
                               time (-t), version (-v), extension (-X)
      --time=WORD            with -l, show time as WORD instead of default
                               modification time: atime or access or use (-u);
                               ctime or status (-c); also use specified time
                               as sort key if --sort=time (newest first)
      --time-style=TIME_STYLE  time/date format with -l; see TIME_STYLE below
  -t                         sort by modification time, newest first
  -T, --tabsize=COLS         assume tab stops at each COLS instead of 8
  -u                         with -lt: sort by, and show, access time;
                               with -l: show access time and sort by name;
                               otherwise: sort by access time, newest first
  -U                         do not sort; list entries in directory order
  -v                         natural sort of (version) numbers within text
  -w, --width=COLS           set output width to COLS.  0 means no limit
  -x                         list entries by lines instead of by columns
  -X                         sort alphabetically by entry extension
  -Z, --context              print any security context of each file
  -1                         list one file per line.  Avoid '\n' with -q or -b
      --help     display this help and exit
      --version  output version information and exit

The SIZE argument is an integer and optional unit (example: 10K is 10*1024).
Units are K,M,G,T,P,E,Z,Y (powers of 1024) or KB,MB,... (powers of 1000).
Binary prefixes can be used, too: KiB=K, MiB=M, and so on.

The TIME_STYLE argument can be full-iso, long-iso, iso, locale, or +FORMAT.
FORMAT is interpreted like in date(1).  If FORMAT is FORMAT1<newline>FORMAT2,
then FORMAT1 applies to non-recent files and FORMAT2 to recent files.
TIME_STYLE prefixed with 'posix-' takes effect only outside the POSIX locale.
Also the TIME_STYLE environment variable sets the default style to use.

Using color to distinguish file types is disabled both by default and
with --color=never.  With --color=auto, ls emits color codes only when
standard output is connected to a terminal.  The LS_COLORS environment
variable can change the settings.  Use the dircolors command to set it.

Exit status:
 0  if OK,
 1  if minor problems (e.g., cannot access subdirectory),
 2  if serious trouble (e.g., cannot access command-line argument).

GNU coreutils online help: <https://www.gnu.org/software/coreutils/>
Full documentation <https://www.gnu.org/software/coreutils/ls>
or available locally via: info '(coreutils) ls invocation'
`

func TestParseLsHelp(t *testing.T) {
	desc, err := ParseHelp([]string{"ls", "--help"}, lsHelp)
	require.Nil(t, err)
	require.NotNil(t, desc)
}

func TestMultipleFlagOccurrences(t *testing.T) {
	fooHelp := `
	usage: foo <flags>
	
	--help show help
	--foo some stuff
	--bar other stuff (see also --foo)
`

	desc, err := ParseHelp([]string{"foo", "--help"}, fooHelp)
	require.Nil(t, err)

	expected := datastore.HelpPage{
		ExecutablePath: "foo",
		Completions: []datastore.Completion{
			{Flag: "--help"},
			{Flag: "--foo"},
			{Flag: "--bar"},
		},
		CheckSum: "1f32a5ebf4758e9a89fca0b0de6aed8761cf6f92",
	}
	require.Equal(t, expected, *desc)
}

func TestJavaStyleInclusion(t *testing.T) {
	fooHelp := `
	usage: foo <flags>
	
	--help show help
	-v --verbose be verbose
	-E --expand expand something
	-T --text expand something
	-a	same as -vET
`

	desc, err := ParseHelp([]string{"foo", "--help"}, fooHelp)
	require.Nil(t, err)

	expected := datastore.HelpPage{
		ExecutablePath: "foo",
		Completions: []datastore.Completion{
			{Flag: "--help"},
			{Flag: "-v"},
			{Flag: "--verbose"},
			{Flag: "-E"},
			{Flag: "--expand"},
			{Flag: "-T"},
			{Flag: "--text"},
			{Flag: "-a"},
		},
		CheckSum: "ab2bce9df3ee2ff08f41d3c8142f5e52ad6c1686",
	}
	require.Equal(t, expected, *desc)
}

func TestJavaStyle(t *testing.T) {
	fooHelp := `
	usage: foo <flags>
	
	-h show help
	-v be verbose
	-E expand something
	-T expand something
	-a same as -vET
`

	desc, err := ParseHelp([]string{"foo", "-h"}, fooHelp)
	require.Nil(t, err)

	expected := datastore.HelpPage{
		ExecutablePath: "foo",
		Completions: []datastore.Completion{
			{Flag: "-h"},
			{Flag: "-v"},
			{Flag: "-E"},
			{Flag: "-T"},
			{Flag: "-a"},
			{Flag: "-vET"}, // it might be
		},
		CheckSum: "5854f53d08357f31d7c9e5d478b12a0a2236c976",
	}
	require.Equal(t, expected, *desc)
}

func TestOptionInVeryBeginningOfLine(t *testing.T) {
	fooHelp := `
usage: foo <flags>
	
-h show help
--foo foo option
`

	desc, err := ParseHelp([]string{"foo", "-h"}, fooHelp)
	require.Nil(t, err)

	expected := datastore.HelpPage{
		ExecutablePath: "foo",
		Completions: []datastore.Completion{
			{Flag: "-h"},
			{Flag: "--foo"}, // it might be
		},
		CheckSum: "da4c610510f4addb1089ae41896d09bfd0ccd790",
	}
	require.Equal(t, expected, *desc)
}
