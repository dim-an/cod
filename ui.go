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

package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"golang.org/x/crypto/ssh/terminal"
)

type UI interface {
	GetKeystroke(choice string) (r rune, err error)
	Styled(style, text string) string
}

func NewUI() UI {
	if terminal.IsTerminal(0) {
		return TerminalUI(0)
	} else {
		return FileUI(0)
	}
}

type FileUI int

func (ui FileUI) GetKeystroke(_ string) (r rune, err error) {
	err = fmt.Errorf("require user interaction but stdin is not a terminal")
	return
}

func (ui FileUI) Styled(_, text string) string {
	return text
}

type TerminalUI int

func (ui TerminalUI) GetKeystroke(choice string) (r rune, err error) {
	mustWrite := func(s string) {
		_, err := os.Stdout.WriteString(s)
		if err != nil {
			panic(err)
		}
	}
	prev, err := terminal.MakeRaw(0)
	if err != nil {
		return
	}
	defer func() {
		_ = terminal.Restore(0, prev)
	}()
	bb := make([]byte, 1)
	for {
		_, err = os.Stdin.Read(bb)
		if err != nil {
			return
		}
		if bb[0] == 3 {
			// Ctrl+C
			mustWrite("\n\r")
			err = fmt.Errorf("interrupt")
			return
		}
		idx := strings.IndexRune(strings.ToLower(choice), unicode.ToLower(rune(bb[0])))
		if idx >= 0 {
			mustWrite(string(bb))
			mustWrite("\n\r")
			r = rune(choice[idx])
			return
		}
	}
}

var styleTable = map[string]string{
	"green": "\033[32m",
}

func (ui TerminalUI) Styled(style, text string) (res string) {
	control, ok := styleTable[style]
	if !ok {
		panic(fmt.Errorf("unknown style: %v", style))
	}
	res = control + text + "\033[0m"
	return
}
