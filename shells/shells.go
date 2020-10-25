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

package shells

import (
	"fmt"
	"github.com/dim-an/cod/datastore"
	"path/filepath"
	"strings"
)

type ShellScriptGenerator interface {
	GetPreamble() []string
	GenerateCompletions(executableName string, completions []datastore.Completion) []string
	ResetCommand(commandName string) []string
}

func NewShellScriptGenerator(shell string) (ShellScriptGenerator, error) {
	switch shell {
	case "bash":
		return &Bash{}, nil
	case "fish":
		return &Fish{}, nil
	case "zsh":
		return &Zsh{}, nil
	default:
		return nil, fmt.Errorf("unknown shell: %v", shell)
	}
}

func isLongOption(opt string) bool {
	return strings.HasPrefix(opt, "--")
}

func isCommand(opt string) bool {
	return !strings.HasPrefix(opt, "-")
}

//
// Zsh
//

type Zsh struct {
}

func (z *Zsh) GetPreamble() (script []string) {
	scriptText := `
__cod_recent_command_zsh=

function __cod_preexec_zsh() {
    __cod_recent_command_zsh="$3"
}

function __cod_postexec_zsh() {
    if [[ "$?" == 0 ]] && [[ -n $__cod_recent_command_zsh ]] ; then
        cod api postexec $$ "$__cod_recent_command_zsh"
        source <(cod api poll-updates $$)
    fi

    return "$old_exit_code"
}

function __cod_add_completions() {
	# -n :: not override existing completions
    compdef -n __cod_complete_zsh "$1"
}

function __cod_clear_completions() {
    if [[ "${_comps[$1]}" == __cod_complete_zsh ]] ; then
        compdef -d "$1"
    fi
}

function __cod_complete_zsh() {
    local c
	local cs
	local c_word
	c_word=$(($CURRENT - 1))
	cs=("${(f)$(cod api complete-words -- $$ "$c_word" "${words[@]}")}")
	for c in "${cs[@]}" ; do
        compadd -- "$c"
    done
	_path_files
}

__cod_clear_completions yt
__cod_add_completions yt

precmd_functions+=("__cod_postexec_zsh")
preexec_functions+=("__cod_preexec_zsh")

cod api attach $$ zsh

`

	script = []string{scriptText}
	return
}

func (z *Zsh) GenerateCompletions(executablePath string, _ []datastore.Completion) (script []string) {
	script = []string{
		fmt.Sprintf(
			"__cod_add_completions %v",
			quoteArg(filepath.Base(executablePath)),
		),
	}
	return
}

func (z *Zsh) ResetCommand(executablePath string) (script []string) {
	script = []string{
		fmt.Sprintf("__cod_clear_completions %v", quoteArg(filepath.Base(executablePath))),
	}
	return
}

//
// Fish
//

type Fish struct {
}

func (f *Fish) GenerateCompletions(executablePath string, completions []datastore.Completion) (shellScript []string) {
	for _, completion := range completions {
		cmd := "complete --command " + executablePath
		if isLongOption(completion.Flag) {
			cmd += " --long-option " + strings.Trim(completion.Flag, "-")
		} else if isCommand(completion.Flag) {
			cmd += " --arguments " + completion.Flag
		} else {
			cmd += " --old-option " + strings.Trim(completion.Flag, "-")
		}
		shellScript = append(shellScript, cmd)
	}
	return
}

func (f *Fish) ResetCommand(commandName string) (shellScript []string) {
	return []string{
		fmt.Sprintf("complete --command %s --erase", commandName),
	}
}

func (f *Fish) GetPreamble() (lines []string) {
	return
}

//
// Bash
//

type Bash struct {
}

func (b *Bash) GenerateCompletions(executablePath string, _ []datastore.Completion) (lines []string) {
	lines = []string{
		fmt.Sprintf(
			"__cod_add_completions %v",
			quoteArg(filepath.Base(executablePath)),
		),
	}
	return
}

func (b *Bash) ResetCommand(executablePath string) (lines []string) {
	lines = []string{
		fmt.Sprintf("__cod_clear_completions %v", quoteArg(filepath.Base(executablePath))),
	}
	return
}

func (b *Bash) GetPreamble() (lines []string) {
	text := `
cod_enable_trace=${cod_enable_trace-false}

__cod_ref_count=0
function __cod_ref_trace() {
	if [ "$((__cod_ref_count))" -eq 0 ] ; then 
		echo "--> inside: ${FUNCNAME[@]}" >&2
		set -x
	fi
    : $((__cod_ref_count++))
}

function __cod_unref_trace() {
	: $((__cod_ref_count--))
	if [ "$__cod_ref_count" -eq 0 ] ; then 
		set +x
	fi
}

function __cod_add_completions() {
	$cod_enable_trace && __cod_ref_trace

	complete -o filenames -o bashdefault -F __cod_complete_bash "$1"

	$cod_enable_trace && __cod_unref_trace
	return 0
}

function __cod_clear_completions() {
	$cod_enable_trace && __cod_ref_trace

	local TO_KEEP
	TO_KEEP=$(complete -p | cod api bash-clean-completions -- "$1")
	eval "$TO_KEEP"

	$cod_enable_trace && __cod_unref_trace
	return 0
}

function __cod_complete_bash() {
	$cod_enable_trace && __cod_ref_trace

	# First we want to get file completions.

	local FILTEROPT
	if [ -z "$2" ] ; then
		# If user trying to complete empty string we want to filter out dot files.
		FILTEROPT='.*'
	fi

	local FILE_COMPLETIONS
	local COD_COMPLETIONS
	# Generate file completions
	readarray -t FILE_COMPLETIONS < <(compgen -f -X "$FILTEROPT" -- "$2")

	# Generate cod completions.
	readarray -t COD_COMPLETIONS < <(cod api complete-words -- $$ "$COMP_CWORD" "${COMP_WORDS[@]}" 2> /dev/null)

	COMPREPLY=("${FILE_COMPLETIONS[@]}" "${COD_COMPLETIONS[@]}")

	local NAME
	local ONLY_EQ=true
	# Now we don't want bash to add trailing space for options that end with '='
	for NAME in "${COD_COMPLETIONS[@]}" ; do
		if [ "${NAME: -1}" != "=" ] ; then
			ONLY_EQ=false
			break
		fi
	done

	if $ONLY_EQ ; then
		compopt -o nospace
	fi

	$cod_enable_trace && __cod_unref_trace
	return 0
}

__cod_postexec_bash_prev_index=
__cod_postexec_bash_first_invocation=1

function __cod_postexec_bash() {
	local old_exit_code="$?"

	$cod_enable_trace && __cod_ref_trace

	while true ; do
		local fc_out command index
		read -ra fc_out <<< $(fc -l -0)

		if [ ${#fc_out[@]} = 0 ] ; then
			break
		fi

		index=${fc_out[0]}

		if [ "$old_exit_code" != 0 ] ; then
			break
		fi

		if [ "$index" = "$__cod_postexec_bash_prev_index" ]; then
			break
		fi
		__cod_postexec_bash_prev_index=$index

		if [ "$__cod_postexec_bash_first_invocation" ] ; then
			__cod_postexec_bash_first_invocation=
			break
		fi

		if [ $? -ne "0" ] ; then
			break
		fi

		command="${fc_out[@]:1}"
		cod api postexec $$ "$command"
		source <(cod api poll-updates $$)
		break
	done

	$cod_enable_trace && __cod_unref_trace
	return $old_exit_code
}

PROMPT_COMMAND="__cod_postexec_bash ; $PROMPT_COMMAND"
`
	lines = []string{text}
	return
}
