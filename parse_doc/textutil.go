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
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode"

	"github.com/dim-an/cod/datastore"
)

type parseContext struct {
	args []string
	text *preparedText
}

func makeParseContext(args []string, helpText string) (ctx parseContext, err error) {
	ctx.text, err = makePreparedText(helpText)
	if err != nil {
		return
	}

	ctx.args = args
	return
}

type parseResult struct {
	description string
	completions []datastore.Completion
}

func completionKey(completion datastore.Completion) string {
	contextBytes, err := json.Marshal(completion.Context)
	if err != nil {
		panic(err)
	}
	return completion.Flag + "\x00" + string(contextBytes)
}

func (res *parseResult) AddCompletion(completion datastore.Completion) {
	key := completionKey(completion)
	for idx := range res.completions {
		if completionKey(res.completions[idx]) == key {
			if res.completions[idx].Description == "" && completion.Description != "" {
				res.completions[idx].Description = completion.Description
			}
			return
		}
	}
	res.completions = append(res.completions, completion)
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
	var extractTree func() lineTree
	extractTree = func() (res lineTree) {
		res = lineTree{
			line:      pt.lines[cur],
			lineBegin: cur,
			lineEnd:   cur,
			children:  nil,
		}
		startIndent := computeIndent(pt.lines[cur])
		for cur++; cur < len(pt.lines); {
			curIndent := computeIndent(pt.lines[cur])
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

// Compute line indent.
// Return -1 if line is visibly empty.
func computeIndent(line string) int {
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

func normalizeDescription(parts ...string) string {
	var cleaned []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, " ")
}

func isMetavar(s string) bool {
	if s == "" {
		return false
	}
	s = strings.Trim(s, "[]{}(),")
	for _, r := range s {
		if r == '_' || r == '-' || ('0' <= r && r <= '9') {
			continue
		}
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func lineDescriptionAfter(line string, start int, stripMetavar bool) string {
	if start >= len(line) {
		return ""
	}
	tail := strings.TrimSpace(line[start:])
	if stripMetavar {
		fields := strings.Fields(tail)
		if len(fields) == 1 && isMetavar(fields[0]) {
			tail = ""
		} else if len(fields) > 1 && isMetavar(fields[0]) {
			tail = strings.TrimSpace(strings.TrimPrefix(tail, fields[0]))
		}
	}
	return tail
}

func collectIndentedDescription(lines []string, start int, baseIndent int) (description []string) {
	for idx := start + 1; idx < len(lines); idx++ {
		line := lines[idx]
		indent := computeIndent(line)
		if indent < 0 {
			break
		}
		if indent <= baseIndent {
			break
		}
		if strings.Contains(line, "-") {
			break
		}
		description = append(description, strings.TrimSpace(line))
	}
	return
}

func extractCommandDescription(text *preparedText) string {
	usageStart := text.FindFirstLine("usage:")
	if usageStart < 0 {
		usageStart = text.FindFirstLine("Usage:")
	}
	if usageStart < 0 {
		return ""
	}

	usageEnd := text.ParagraphEnd(usageStart)
	var usageDescriptions []string
	for idx := usageStart + 1; idx < usageEnd; idx++ {
		line := strings.TrimSpace(text.lines[idx])
		if line != "" && !strings.HasPrefix(line, "[") && computeIndent(text.lines[idx]) == 0 {
			usageDescriptions = append(usageDescriptions, line)
		}
	}
	if len(usageDescriptions) > 0 {
		return normalizeDescription(usageDescriptions...)
	}

	for idx := usageEnd + 1; idx < len(text.lines); idx++ {
		line := strings.TrimSpace(text.lines[idx])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(line, "-") ||
			computeIndent(text.lines[idx]) > 0 ||
			strings.HasSuffix(lower, "arguments:") ||
			strings.HasSuffix(lower, "options:") ||
			strings.HasSuffix(lower, "commands:") {
			return ""
		}
		end := text.ParagraphEnd(idx)
		return normalizeDescription(text.lines[idx:end]...)
	}
	return ""
}
