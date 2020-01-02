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
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func BashRemoveCompletions(appBase string, reader io.Reader) (completions []string, err error) {
	buffedReader := bufio.NewReader(reader)

	commandsToKeep := make(map[string][]string)
	commandsToRemove := make(map[string]bool)

	for {
		var line string
		line, err = buffedReader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		if errors.Is(err, io.EOF) {
			err = nil
			break
		} else if err != nil {
			return
		}

		tokens, err := Tokenize(line)
		if err != nil || len(tokens) == 0 || tokens[0].Decoded != "complete" {
			// We are playing safe and if we don't understand the line we just return it as is
			continue
		}

		curCommandName := tokens[len(tokens)-1].Decoded
		if filepath.Base(curCommandName) != appBase {
			continue
		}

		toRemove := false
	loop:
		for _, t := range tokens {
			switch {
			case strings.HasPrefix(t.Decoded, "__cod_") || t.Decoded == "_minimal":
				toRemove = true
				break loop
			case t.Decoded == "-D":
				toRemove = false
				break loop
			}
		}
		if toRemove {
			commandsToRemove[curCommandName] = true
		} else {
			commandsToKeep[curCommandName] = append(commandsToKeep[curCommandName], line)
		}
	}

	var sortedCommandsToRemove []string
	for commandName := range commandsToRemove {
		sortedCommandsToRemove = append(sortedCommandsToRemove, commandName)
	}
	sort.Strings(sortedCommandsToRemove)

	for _, commandName := range sortedCommandsToRemove {
		completions = append(completions, fmt.Sprintf("complete -r %s", quoteArg(commandName)))
		completions = append(completions, commandsToKeep[commandName]...)
	}

	return
}
