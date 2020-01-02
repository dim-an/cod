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

import "strings"

func Quote(args []string) string {
	var quoted []string
	for _, a := range args {
		quoted = append(quoted, quoteArg(a))
	}
	return strings.Join(quoted, " ")
}

func quoteArg(arg string) string {
	argBytes := []byte(arg)

	needQuoting := false
	hasSingleQuote := false
	for _, c := range argBytes {
		switch {
		case c == '\'':
			needQuoting = true
			hasSingleQuote = true
		case checkNeedQuoting(c):
			needQuoting = true
		}
	}

	if !needQuoting {
		return arg
	}

	sb := strings.Builder{}
	if hasSingleQuote {
		sb.WriteByte('"')
		for _, c := range argBytes {
			if checkNeedQuoting(c) {
				sb.WriteByte('\\')
			}
			sb.WriteByte(c)
		}
		sb.WriteByte('"')
	} else {
		sb.WriteByte('\'')
		sb.WriteString(arg)
		sb.WriteByte('\'')
	}

	return sb.String()
}

func checkNeedQuoting(c byte) bool {
	switch c {
	case '\'', '|', '&', ';', '<', '>', '(', ')',
		'$', '`', '\\', '"', ' ', '\t', '\n',
		'*', '?', '[', ']', '#', '~', '=', '%':
		return true
	default:
		return false
	}
}
