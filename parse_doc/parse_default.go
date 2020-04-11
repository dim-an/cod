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
	"cod/datastore"
	"regexp"
	"strings"
)

var flagRegexp = regexp.MustCompile(`(-[-[:word:]]+(?:=?))`)
var allFlagsRegexp = regexp.MustCompile(`^\s+(?:(-[-[:word:]]+)(?:[ =A-Z_\[\]]*)?[, \t]*)+`)
var subCommandRegexp = regexp.MustCompile(`^\s+([-[:word:]]+)`)

type defaultParser struct{}

func makeDefaultParser() HelpParser {
	return defaultParser{}
}

func (defaultParser) Name() string {
	return "default"
}

func (defaultParser) Parse(context parseContext) (res *parseResult, err error) {
	res = &parseResult{}

	var completions []datastore.Completion
	for _, line := range context.text.lines {
		m := allFlagsRegexp.FindString(line)
		if len(m) == 0 {
			continue
		}
		flagsMatch := flagRegexp.FindAllString(m, -1)
		for _, match := range flagsMatch {
			completions = append(completions, datastore.Completion{Flag: match})
		}
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
		Outer = iota
		FirstLineInside = iota
		Inside = iota
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
				m := subCommandRegexp.FindStringSubmatch(line)
				if m == nil {
					// Unexpected line without command going back to safety.
					state = Outer
					continue
				}
				subCommand := m[1]
				completions = append(completions, datastore.Completion{Flag: subCommand})
			} else if indent < currentParagraphIndent {
				state = Outer
			} // else if indent > currentParagraphIndent { continue }
		}
		prevIndent = indent
	}

	res.completions = completions
	return
}
