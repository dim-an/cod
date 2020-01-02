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
	"cod/shells/asciitable"
	"cod/util"
	"fmt"
	"io"
	"strings"
)

type Token struct {
	Decoded  string
	Original string

	// Position of the token in the original string
	OrigBegin int
	OrigEnd   int

	// Indicator of reserved word (e.g. `for`, `while`, `case` etc).
	// NB. Control operators such as `&&` are not reserved word
	IsReserved bool

	// Indicator of broken token
	IsBroken bool

	// Indicator of scary token (e.g. '`').
	// Commands with scary token are to complex for cod to reason about.
	IsScary bool
}

func Tokenize(command string) (toks []Token, err error) {
	t := tokenizer{
		command: command,
		reader: positionedScanner{
			scanner: strings.NewReader(command),
			pos:     0,
		},
	}
	err = t.tokenize()
	if err != nil {
		return
	}
	toks = t.result
	return
}

// http://man7.org/linux/man-pages/man1/bash.1.html#RESERVED_WORDS
var reservedWords = map[string]bool{
	"!":        true,
	"case":     true,
	"coproc":   true,
	"do":       true,
	"done":     true,
	"elif":     true,
	"else":     true,
	"esac":     true,
	"fi":       true,
	"for":      true,
	"function": true,
	"if":       true,
	"in":       true,
	"select":   true,
	"then":     true,
	"until":    true,
	"while":    true,
	"{":        true,
	"}":        true,
	"time":     true,
	"[[":       true,
	"]]":       true,
}

type positionedScanner struct {
	scanner io.ByteScanner
	pos     int
}

func (s *positionedScanner) ReadByte() (b byte, err error) {
	b, err = s.scanner.ReadByte()
	if err != nil {
		s.pos += 1
	}
	return
}

func (s *positionedScanner) UnreadByte() (err error) {
	err = s.scanner.UnreadByte()
	if err != nil {
		s.pos -= 1
	}
	return
}

func (s *positionedScanner) GetPosition() int {
	return s.pos
}

type tokenizer struct {
	command  string
	reader   positionedScanner
	builder  strings.Builder
	result   []Token
	curBegin int
	nonEmpty bool
	scary    bool
	broken   bool
}

// http://man7.org/linux/man-pages/man1/bash.1.html#QUOTING
func (t *tokenizer) tokenize() error {
	for {
		c, err := t.reader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			// If we are working with strings we don't expect any exceptions here and can panic if the error occurred.
			// But in the future we might want to use other ByteReaders so we handle error in ordinary way.
			return err
		}
		switch c {
		case '\\':
			c, err = t.reader.ReadByte()
			if err == io.EOF {
				t.broken = true
				t.emitWord()
				return nil
			} else if err != nil {
				return err
			}
			if c == '\n' {
				// ignore it
			} else {
				t.builder.WriteByte(c)
			}
		case '\n':
			t.emitWord()
			t.builder.WriteByte(c)
			t.emitWord()
		case '\'':
			err = t.parseSingleQuote()
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
		case '"':
			err = t.parseDoubleQuote()
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
		case ' ', '\t':
			t.emitWord()
		case '$':
			c, err = t.reader.ReadByte()
			switch {
			case err == io.EOF:
				t.builder.WriteByte('$')
			case err != nil:
				return err
			case c == '\'' || c == '"':
				err = t.parseAsciiString(c)
				if err != nil {
					return err
				}
			default:
				t.builder.WriteByte('$')
				err = t.reader.UnreadByte()
				util.VerifyPanic(err)
			}
		case '|', '&', ';', '(', ')', '<', '>', '`':
			t.emitWord()

			t.scary = true
			t.builder.WriteByte(c)
			t.emitWord()
		default:
			t.builder.WriteByte(c)
		}
	}
	t.emitWord()
	return nil
}

func (t *tokenizer) parseSingleQuote() error {
	t.nonEmpty = true
	for {
		c, err := t.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				t.broken = true
				t.emitWord()
			}
			return err
		}
		if c == '\'' {
			break
		}
		t.builder.WriteByte(c)
	}
	return nil
}

//
func (t *tokenizer) parseDoubleQuote() error {
	t.nonEmpty = true
	for {
		c, err := t.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				t.broken = true
				t.emitWord()
			}
			return err
		}
		switch c {
		case '"':
			return nil
		case '\\':
			c, err = t.reader.ReadByte()
			if err != nil {
				if err == io.EOF {
					t.broken = true
					t.emitWord()
				}
				return err
			}
			switch c {
			case '$', '`', '"', '\\':
				t.builder.WriteByte(c)
			case '\n':
				// do nothing
			default:
				t.builder.WriteByte('\\')
				t.builder.WriteByte(c)
			}
		default:
			t.builder.WriteByte(c)
		}
	}
}

func (t *tokenizer) emitWord() {
	curPos := t.reader.GetPosition()
	if t.builder.Len() > 0 || t.nonEmpty {
		data := t.builder.String()
		_, isReserved := reservedWords[data]
		t.result = append(t.result, Token{
			Decoded:    data,
			OrigBegin:  t.curBegin,
			OrigEnd:    curPos,
			IsReserved: isReserved,
			IsBroken:   t.broken,
			IsScary:    t.scary,
		})
		t.builder.Reset()
	}
	t.curBegin = curPos
	t.nonEmpty = false
	t.scary = false
	t.broken = false
}

func (t *tokenizer) parseAsciiString(end byte) (err error) {
	if end == '\\' {
		panic(fmt.Errorf("`\\` cannot be the end of ascii string"))
	}
	var c byte
	for {
		c, err = t.reader.ReadByte()
		switch {
		case err != nil:
			if err == io.EOF {
				t.broken = true
				t.emitWord()
			}
			return
		case c == end:
			return
		case c == '\\':
			err = t.parseAsciiChar()
			if err != nil {
				return
			}
		default:
			t.builder.WriteByte(c)
		}
	}
}

func (t *tokenizer) parseAsciiChar() (err error) {
	b, err := t.reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			t.broken = true
			t.emitWord()
		}
		return
	}

	switch b {
	case '\n':
		// do nothing
	case 'a':
		t.builder.WriteByte(asciitable.BEL)
	case 'b':
		t.builder.WriteByte(asciitable.BS)
	case 'e', 'E':
		t.builder.WriteByte(asciitable.ESC)
	case 'f':
		t.builder.WriteByte(asciitable.FF)
	case 'n':
		t.builder.WriteByte(asciitable.LF)
	case 'r':
		t.builder.WriteByte(asciitable.CR)
	case 't':
		t.builder.WriteByte(asciitable.TAB)
	case 'v':
		t.builder.WriteByte(asciitable.VT)
	case '\\':
		t.builder.WriteByte('\\')
	case '\'':
		t.builder.WriteByte('\'')
	case '"':
		t.builder.WriteByte('"')
	case '?':
		t.builder.WriteByte('?')
	case '0', '1', '2', '3', '4', '5', '6', '7', '8':
		fallthrough
	case 'x', 'u', 'U', 'c':
		err = util.ErrNotImplemented
		return
	}
	return
}
