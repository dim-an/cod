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
	"bufio"
	"cod/datastore"
	"errors"
	"io"
	"strings"
	"unicode"
)

type parseContext struct {
	executablePath string
	text           *preparedText
}

func makeParseContext(executablePath, helpText string) (ctx parseContext, err error) {
	ctx.text, err = makePreparedText(helpText)
	if err != nil {
		return
	}

	ctx.executablePath = executablePath
	return
}

type parseResult struct {
	completions []datastore.Completion
}

type preparedText struct {
	lines []string
}

func makePreparedText(text string) (prepared *preparedText, err error) {
	reader := bufio.NewReader(strings.NewReader(text))

	// 1. Split by lines
	var lines []string
	for {
		var line string
		line, err = reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			err = nil
			break
		} else if err != nil {
			return
		}
		lines = append(lines, line[:len(line)-1])
	}

	prepared = &preparedText{
		lines: lines,
	}
	return
}

func (pt *preparedText) FindFirstLine(pattern string) int {
	for idx, line := range pt.lines {
		if strings.Contains(line, pattern) {
			return idx
		}
	}
	return -1
}

//
// Find first empty line starting from line `start'.
func (pt *preparedText) ParagraphEnd(start int) int {
	cur := start
	for ; cur < len(pt.lines); cur++ {
		if strings.TrimSpace(pt.lines[cur]) == "" {
			return cur
		}
	}
	return cur
}

type lineTree struct {
	line      string
	lineBegin int
	lineEnd   int
	children  []lineTree
}

func (pt *preparedText) FindIndentedParagraph(pattern string, startLine int) *lineTree {
	var cur int
	calcIndent := func(line string) int {
		indent := 0
		for _, r := range line {
			if unicode.IsSpace(r) {
				indent += 1
			} else {
				return indent
			}
		}
		return -1
	}

	var extractTree func() lineTree
	extractTree = func() (res lineTree) {
		res = lineTree{
			line:      pt.lines[cur],
			lineBegin: cur,
			lineEnd:   cur,
			children:  nil,
		}
		startIndent := calcIndent(pt.lines[cur])
		for cur++; cur < len(pt.lines); {
			curIndent := calcIndent(pt.lines[cur])
			if curIndent > startIndent {
				res.children = append(res.children, extractTree())
			} else {
				break
			}
		}
		res.lineEnd = cur
		return
	}

	cur = startLine
	for ; cur < len(pt.lines); cur++ {
		if strings.Contains(pt.lines[cur], pattern) {
			res := extractTree()
			return &res
		}
	}

	return nil
}
