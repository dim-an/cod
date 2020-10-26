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
	"path"
	"regexp"
	"strings"

	"github.com/dim-an/cod/datastore"
)

// NB. reminder (?: ... ) is a non-capturing group
var flagRegexp = regexp.MustCompile(`(?:^|[\s\[|])(-[-[:word:]]+(?:=?))`)
var javaStyleFlagRegexp = regexp.MustCompile(`^-[[:word:]]{2,}$`)

var indentedSubCommandRegexp = regexp.MustCompile(`^\s+([-[:word:]]+)`)
var subCommandRegexp = regexp.MustCompile(`^[[:word:]][-[:word:]]*$`)
var wordRegexp = regexp.MustCompile(`\S+`)

type defaultParser struct{}

func makeDefaultParser() HelpParser {
	return defaultParser{}
}

func (defaultParser) Name() string {
	return "default"
}

func isJavaStyleFlag(flag string) bool {
	return javaStyleFlagRegexp.MatchString(flag)
}

func parseUsageSubCommand(args []string, text *preparedText) (res []string) {
	var words []string
	foundUsage := false

	// First of all we split the first paragraph of help into words
	// and make sure that first word is `usage`
	for _, line := range text.lines {
		curWords := wordRegexp.FindAllString(line, -1)
		if curWords == nil {
			break
		}
		if !foundUsage {
			if strings.ToLower(curWords[0]) != "usage:" &&
				strings.ToLower(curWords[0]) != "usage" {
				return
			}
			foundUsage = true
			// Don't need `usage word`
			words = append(words, curWords[1:]...)
			continue
		}
		words = append(words, curWords...)
	}

	// Then we make sure that the first word after usage is name of application (we check only basename).
	executableBase := path.Base(args[0])
	if words == nil || len(words) < 1 {
		return
	}

	if path.Base(words[0]) != executableBase {
		return
	}

	argsIdx := 1
	wordIdx := 1

	// We iterate over words of usage paragraph until we find the word that can't be sub-command.
	// We check each word in actual command line to double check that it's actual sub-command used.
outerLoop:
	for ; wordIdx < len(words); wordIdx += 1 {
		w := words[wordIdx]
		if !subCommandRegexp.MatchString(w) {
			wordIdx += 1
			break
		}
		for ; argsIdx < len(args); argsIdx += 1 {
			if w == args[wordIdx] {
				res = append(res, w)
				continue outerLoop
			}
		}
		res = nil
		break
	}

	if res == nil {
		return
	}

	// We check that usage paragraph doesn't contain any other application name word.
	// If it does we are probably dealing with help that describes multiple sub-commands.
	// We don't support them.
	for _, w := range words[wordIdx:] {
		if w == executableBase {
			res = nil
			break
		}
	}

	return
}

func (defaultParser) Parse(context parseContext) (res *parseResult, err error) {
	res = &parseResult{}

	flagContext := datastore.FlagContext{
		SubCommand: parseUsageSubCommand(context.args, context.text),
	}

	var completions []datastore.Completion
	var discoveredFlagMap = make(map[string]bool)
	var discoveredFlags []string
	for _, line := range context.text.lines {
		flagsMatch := flagRegexp.FindAllStringSubmatch(line, -1)
		for _, match := range flagsMatch {
			flag := match[1]
			if !discoveredFlagMap[flag] {
				discoveredFlags = append(discoveredFlags, flag)
				discoveredFlagMap[flag] = true
			}
		}
	}

	// Sometimes we find examples of merged single letter options like `-xzf` (== -x -z -f) in help messages.
	// We want to distinguish them from java style-options like `-server`.
	// We want to guess if we are dealing with gnu style or java style.
	isGnuLike := false
	for _, flag := range discoveredFlags {
		if strings.HasPrefix(flag, "--") {
			isGnuLike = true
			break
		}
	}

	for _, flag := range discoveredFlags {
		if isGnuLike && isJavaStyleFlag(flag) {
			continue
		}
		completions = append(completions, datastore.Completion{Flag: flag, Context: flagContext})
	}

	// Now we are going to search for sub-commands.
	// Idea is following
	//   1. We are looking for string that ends with `commands:` (we ignore case)
	//   2. Then we check indentation of the next line
	//   3. First word of the line that has same indentation as first line is sub-command.
	//   4. We skip the line if line indent is bigger than indent of the first line
	//      (most likely this is continuation of help)
	//   5. We stop when we find empty line or line that has indent less than indent of the first line.
	const (
		Outer           = iota
		FirstLineInside = iota
		Inside          = iota
	)
	var state = Outer
	var prevIndent = -1
	var currentParagraphIndent = 0
	for _, line := range context.text.lines {
		var indent = computeIndent(line)
		switch state {
		case Outer:
			line = strings.ToLower(line)
			if indent == 0 && strings.HasSuffix(line, "commands:") {
				state = FirstLineInside
			}
		case FirstLineInside:
			if indent <= prevIndent {
				state = Outer
				continue
			}
			currentParagraphIndent = indent
			state = Inside
			fallthrough
		case Inside:
			if indent == currentParagraphIndent {
				m := indentedSubCommandRegexp.FindStringSubmatch(line)
				if m == nil {
					// Unexpected line without command going back to safety.
					state = Outer
					continue
				}
				subCommand := m[1]
				completions = append(completions, datastore.Completion{Flag: subCommand, Context: flagContext})
			} else if indent < currentParagraphIndent {
				state = Outer
			} // else if indent > currentParagraphIndent { continue }
		}
		prevIndent = indent
	}

	res.completions = completions
	return
}
