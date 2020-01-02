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
	"errors"
	"fmt"
	"strings"
)

var ErrCommandNotSimple = errors.New("")

func errCommandNotSimple(reason, cmd string) error {
	return fmt.Errorf("%w%v: %q", ErrCommandNotSimple, reason, cmd)
}

func isVariableAssignment(s string) bool {
	return strings.ContainsRune(s, '=')
}

func ParseSimpleCommand(cmd string) (env, args []string, err error) {
	tokens, err := Tokenize(cmd)
	if err != nil {
		return
	}

	for _, t := range tokens {
		if t.IsScary {
			err = errCommandNotSimple("command is not simple", cmd)
			args = nil
			env = nil
			return
		}
		if isVariableAssignment(t.Decoded) && len(args) == 0 {
			env = append(env, t.Decoded)
		} else {
			args = append(args, t.Decoded)
		}
	}

	if len(args) == 0 {
		err = errCommandNotSimple("nothing to run", cmd)
		args = nil
		env = nil
		return
	}

	return
}
