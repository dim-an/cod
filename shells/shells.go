// Copyright 2020-2021 Dmitry Ermolov
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
	"path/filepath"

	"github.com/dim-an/cod/datastore"
)

type ShellScriptGenerator interface {
	GetPreamble() []string
	GenerateCompletions(executableName string, completions []datastore.Completion) []string
	ResetCommand(executablePath string) []string
}

func NewShellScriptGenerator(shell string, codBinary string) (ShellScriptGenerator, error) {
	switch shell {
	case "bash":
		return &Bash{
			codBinary,
		}, nil
	case "fish":
		return &Fish{
			codBinary,
		}, nil
	case "zsh":
		return &Zsh{
			codBinary,
		}, nil
	default:
		return nil, fmt.Errorf("unknown shell: %v", shell)
	}
}

//
// Zsh
//

type Zsh struct {
	codCommandPath string
}

func (z *Zsh) GetPreamble() (script []string) {
	codBinaryVar := "__COD_BINARY=" + quoteArg(z.codCommandPath)
	scriptText := `
__cod_recent_command_zsh=

function __cod_preexec_zsh() {
    __cod_recent_command_zsh="$3"
}

function __cod_postexec_zsh() {
    if [[ "$?" == 0 ]] && [[ -n $__cod_recent_command_zsh ]] ; then
        command $__COD_BINARY api postexec -- $$ "$__cod_recent_command_zsh"
        source <(command $__COD_BINARY api poll-updates -- $$)
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
	cs=("${(f)$(command $__COD_BINARY api complete-words -- $$ "$c_word" "${words[@]}")}")
	for c in "${cs[@]}" ; do
        compadd -- "$c"
    done
	_path_files
}
precmd_functions+=("__cod_postexec_zsh")
preexec_functions+=("__cod_preexec_zsh")

command $__COD_BINARY api attach -- $$ zsh
`

	script = []string{
		codBinaryVar,
		scriptText,
	}
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
	codCommandPath string
}

func (f *Fish) GenerateCompletions(executablePath string, _ []datastore.Completion) (shellScript []string) {
	shellScript = []string{
		fmt.Sprintf("complete --command %s --arguments '(__cod_complete_fish)'",
			quoteArg(filepath.Base(executablePath))),
	}
	return
}

func (f *Fish) ResetCommand(executablePath string) (shellScript []string) {
	return []string{
		fmt.Sprintf("complete --command %s --erase",
			quoteArg(filepath.Base(executablePath))),
	}
}

func (f *Fish) GetPreamble() (lines []string) {
	lines = []string{
		fmt.Sprintf("set -g __COD_BINARY %v", quoteArg(f.codCommandPath)),
		`
function __cod_complete_fish
    set -l words (commandline --current-process --tokenize --cut-at-cursor)
    set -l cword (count $words)
    set -l words $words (commandline --current-token --cut-at-cursor)
    set -l compreply (command $__COD_BINARY api complete-words -- %self "$cword" $words)
    for entry in $compreply
        echo $entry
    end
    return 0
end

function __fish_cod_get_completions
end

function __cod_postexec_fish --on-event fish_postexec
	set -l cmd "$argv[1]"
	if test -n "$cmd" -a "$status" -eq 0
		command $__COD_BINARY api postexec -- %self "$cmd"
	end
	command $__COD_BINARY api poll-updates -- %self | source
end

cod api attach -- %self fish
`,
	}
	return
}

//
// Bash
//

type Bash struct {
	codCommandPath string
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
	codBinaryVar := fmt.Sprintf("__COD_BINARY=%v", quoteArg(b.codCommandPath))
	scriptText := `
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
	TO_KEEP=$(complete -p | command $__COD_BINARY api bash-clean-completions -- "$1")
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
	readarray -t COD_COMPLETIONS < <(command $__COD_BINARY api complete-words -- $$ "$COMP_CWORD" "${COMP_WORDS[@]}" 2> /dev/null)

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
		command $__COD_BINARY api postexec -- $$ "$command"
		source <(command $__COD_BINARY api poll-updates -- $$)
		break
	done

	$cod_enable_trace && __cod_unref_trace
	return $old_exit_code
}

PROMPT_COMMAND="__cod_postexec_bash;$PROMPT_COMMAND"
`
	lines = []string{
		codBinaryVar,
		scriptText,
	}
	return
}
