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
	"github.com/stretchr/testify/require"
	"testing"
)

type treeNode struct {
	childDepth int
	line       string
}

func walkLineTree(tree *lineTree) (res []treeNode) {
	var walk func(int, *lineTree)
	walk = func(initialDepth int, tree *lineTree) {
		res = append(res, treeNode{
			childDepth: initialDepth,
			line:       tree.line,
		})
		for i := range tree.children {
			walk(initialDepth+1, &tree.children[i])
		}
	}
	walk(0, tree)
	return
}

func TestFindIndentedParagraph(t *testing.T) {
	makeFlattened := func(text, paragraph string) []treeNode {
		prepared, err := makePreparedText(text)
		require.NoError(t, err)

		tree := prepared.FindIndentedParagraph(paragraph, 0)
		require.NotNil(t, tree)
		return walkLineTree(tree)
	}

	text := `
foo bar
even more foo bar

optional arguments:
  -h, --help            show this help message and exit
  --version             show program's version number and exit

foo bar
`
	require.Equal(t,
		[]treeNode{
			{0, "optional arguments:"},
			{1, "  -h, --help            show this help message and exit"},
			{1, "  --version             show program's version number and exit"},
		},
		makeFlattened(text, "arguments:"))

	text = `
there comes list:
  - foo
   - bar
 - baz
 - qux
`

	require.Equal(t,
		[]treeNode{
			{0, "there comes list:"},
			{1, "  - foo"},
			{2, "   - bar"},
			{1, " - baz"},
			{1, " - qux"},
		},
		makeFlattened(text, "list:"))

	text = `
there comes list:
  - foo
   - bar

 - baz
 - qux
`

	require.Equal(t,
		[]treeNode{
			{0, "there comes list:"},
			{1, "  - foo"},
			{2, "   - bar"},
		},
		makeFlattened(text, "list:"))
}
