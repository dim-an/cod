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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dim-an/cod/datastore"
)

type usageLexer struct {
	curToken    string
	curIsSyntax bool
	curLine     string
	restLines   []string
	valid       bool
	err         error
}

func makeUsageLexer(usage []string) usageLexer {
	cur := strings.SplitAfterN(usage[0], "usage:", 2)[1]
	res := usageLexer{
		curLine:   cur,
		restLines: usage[1:],
		valid:     true,
	}
	return res
}

var tokenRe = regexp.MustCompile("^[-_/.a-zA-Z0-9]+")

func (l *usageLexer) cutToken(length int, isSyntax bool) {
	l.curToken = l.curLine[:length]
	l.curLine = l.curLine[length:]
	l.curIsSyntax = isSyntax
}

func (l *usageLexer) Valid() bool {
	return l.valid
}

func (l *usageLexer) abort(err error) {
	l.err = err
	l.valid = false
}

func (l *usageLexer) Next() bool {
	l.skipSpaces()
	if !l.valid {
		return false
	}
	switch l.curLine[0] {
	case '{', '}', '|', '[', ']', ',':
		l.cutToken(1, true)
		return true
	}

	t := tokenRe.FindString(l.curLine)
	if len(t) > 0 {
		l.cutToken(len(t), false)
		return true
	}
	l.abort(fmt.Errorf("cannot tokenize: %v", l.curLine))
	return false
}

func (l *usageLexer) Token() string {
	return l.curToken
}

func (l *usageLexer) TokenIsSyntax() bool {
	return l.curIsSyntax
}

func (l *usageLexer) Err() error {
	return l.err
}

func (l *usageLexer) skipSpaces() {
	for l.valid {
		l.curLine = strings.TrimSpace(l.curLine)
		if len(l.curLine) > 0 {
			return
		}
		if len(l.restLines) > 0 {
			l.curLine = l.restLines[0]
			l.restLines = l.restLines[1:]
		} else {
			l.valid = false
		}
	}
}

type argparseUsage struct {
	applicationName         string
	positionalArgumentNames []string
	positionalArguments     []string
	flagContext             datastore.FlagContext
}

func parseArgparseUsage(lexer *usageLexer) (usage argparseUsage, err error) {
	// First token should be application name
	if !lexer.Next() {
		err = lexer.Err()
		if err == nil {
			err = fmt.Errorf("bad usage: cannot find application name")
		}
		return
	}
	usage.applicationName = lexer.Token()

	// Then should go tokens of sub-command up to first '['
	lexer.Next()
	usage.flagContext.Framework = "argparse"
	for {
		if !lexer.Valid() {
			err = lexer.Err()
			if err == nil {
				err = fmt.Errorf("cannot find [-h] in usage")
			}
			return
		}
		if lexer.TokenIsSyntax() {
			break
		}
		usage.flagContext.SubCommand = append(usage.flagContext.SubCommand, lexer.Token())
		lexer.Next()
	}

	// now a bunch of groups should follow
	parseOptionalGroup := func() error {
		lexer.Next()
	loop:
		for {
			if !lexer.Valid() {
				err = lexer.Err()
				if err == nil {
					err = fmt.Errorf("unexpected end of usage while parsing optional group")
				}
				return err
			}
			switch lexer.Token() {
			case "[":
				return fmt.Errorf("optional group cannot be nested")
			case "]":
				break loop
			}
			lexer.Next()
		}
		lexer.Next()
		return nil
	}
	parseChoiceGroup := func() error {
		lexer.Next()
	loop:
		for {
			if !lexer.Valid() {
				err = lexer.Err()
				if err == nil {
					err = fmt.Errorf("unexpected end of usage while parsing optional group")
				}
				return err
			}
			switch {
			case !lexer.TokenIsSyntax():
				usage.positionalArguments = append(usage.positionalArguments, lexer.Token())
			case lexer.Token() == ",":
				// do nothing
			case lexer.Token() == "{":
				return fmt.Errorf("choice group cannot be nested")
			case lexer.Token() == "}":
				break loop
			default:
				return fmt.Errorf("unexpected token %q in choice group", lexer.Token())
			}
			lexer.Next()
		}
		lexer.Next()
		return nil
	}

	parseGroup := func() error {
		if !lexer.TokenIsSyntax() {
			usage.positionalArgumentNames = append(usage.positionalArgumentNames, lexer.Token())
			lexer.Next()
		} else {
			switch lexer.Token() {
			case "{":
				err = parseChoiceGroup()
				if err != nil {
					return err
				}
			case "[":
				err = parseOptionalGroup()
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("bad token in usage: %q", lexer.curLine)
			}
		}
		return lexer.Err()
	}

	for lexer.Valid() {
		err = parseGroup()
		if err != nil {
			return
		}
	}
	err = lexer.Err()
	return
}

var argWord = "[_a-zA-Z0-9][-_a-zA-Z0-9]*"

var flagsLineRe = regexp.MustCompile(fmt.Sprintf("^ +-{1,2}%s", argWord))
var flagRe = regexp.MustCompile(fmt.Sprintf("-{1,2}%s", argWord))
var argRe = regexp.MustCompile(fmt.Sprintf("^\\s*(%s)(,|\\s|$)", argWord))
var unnamedSequenceRe = regexp.MustCompile(fmt.Sprintf("^\\s*\\{%s(,%s)*\\}$", argWord, argWord))

func tryParseFlagsParagraph(par *lineTree, usage *argparseUsage, res *parseResult) bool {
	if len(par.children) == 0 || !flagsLineRe.MatchString(par.children[0].line) {
		return false
	}

	for idx := range par.children {
		line := par.children[idx].line
		allFlags := flagRe.FindAllString(line, -1)
		for _, flag := range allFlags {
			res.completions = append(res.completions, datastore.Completion{
				Flag:    flag,
				Context: usage.flagContext,
			})
		}
	}
	return true
}

func extractPositionalArgs(par *lineTree, usage *argparseUsage, res *parseResult) bool {
	var completions []datastore.Completion
	for idx := range par.children[0].children {
		line := par.children[0].children[idx].line
		arg := argRe.FindStringSubmatch(line)[1]
		if len(arg) == 0 {
			return false
		}
		completions = append(completions, datastore.Completion{
			Flag:    arg,
			Context: usage.flagContext,
		})
	}
	res.completions = append(res.completions, completions...)
	return true
}

func tryParseNamedPositionalParagraph(par *lineTree, usage *argparseUsage, res *parseResult) bool {
	if len(par.children) != 1 {
		return false
	}
	var argName string
	{
		match := argRe.FindStringSubmatch(par.children[0].line)
		if len(match) == 0 {
			return false
		}
		argName = match[1]
	}
	isKnownArgName := false
	for idx := range usage.positionalArgumentNames {
		if usage.positionalArgumentNames[idx] == argName {
			isKnownArgName = true
			break
		}
	}
	if !isKnownArgName {
		return false
	}

	return extractPositionalArgs(par, usage, res)
}

func tryParseUnnamedPositionalParagraph(par *lineTree, usage *argparseUsage, res *parseResult) bool {
	if len(par.children) != 1 {
		return false
	}
	if !unnamedSequenceRe.MatchString(par.children[0].line) {
		return false
	}
	return extractPositionalArgs(par, usage, res)
}

type argparseParser struct{}

func makeArgparseParser() HelpParser {
	return argparseParser{}
}

func (argparseParser) Name() string {
	return "argparse"
}

func (argparseParser) Parse(context parseContext) (res *parseResult, err error) {
	usageStartIndex := context.text.FindFirstLine("usage:")
	if usageStartIndex < 0 {
		err = fmt.Errorf("cannot find usage")
		return
	}
	if usageStartIndex != 0 {
		err = fmt.Errorf("usage is not at the beginning, doesn't look like argparse")
	}

	usageEndIndex := context.text.ParagraphEnd(usageStartIndex)
	usageLexer := makeUsageLexer(context.text.lines[usageStartIndex:usageEndIndex])

	usage, err := parseArgparseUsage(&usageLexer)
	if err != nil {
		err = fmt.Errorf("error parsing usage: %v", err)
		return
	}

	if filepath.Base(usage.applicationName) != filepath.Base(context.args[0]) {
		err = fmt.Errorf("application in usage doesn't match provided application")
		return
	}

	result := &parseResult{}
	for start := 0; start < len(context.text.lines); {
		par := context.text.FindIndentedParagraph("arguments:", start)
		if par == nil {
			break
		}
		start = par.lineEnd
		if len(par.children) == 0 {
			continue
		}
		switch {
		case tryParseFlagsParagraph(par, &usage, result):
		case tryParseNamedPositionalParagraph(par, &usage, result):
		case tryParseUnnamedPositionalParagraph(par, &usage, result):
		}
	}

	res = result
	return
}
