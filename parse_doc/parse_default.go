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
)

var flagRegexp = regexp.MustCompile(`(-[-[:word:]]+(?:=?))`)
var allFlagsRegexp = regexp.MustCompile(`^ \s+(?:(-[-[:word:]]+)(?:[ =A-Z_\[\]]*)?[, \t]*)+`)

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
	res.completions = completions
	return
}
