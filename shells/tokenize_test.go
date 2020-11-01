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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenizeStrings(t *testing.T) {
	check := func(cmd string, expectedArgs []string) {
		var expectedBroken []bool
		for range expectedArgs {
			expectedBroken = append(expectedBroken, false)
		}

		tokens, err := Tokenize(cmd)
		var args []string
		var broken []bool
		for _, t := range tokens {
			args = append(args, t.Decoded)
			broken = append(broken, t.IsBroken)
		}

		require.Nil(t, err)
		require.Equal(t, expectedArgs, args)
		require.Equal(t, expectedBroken, broken)
	}

	// word splitting
	check(`echo foo bar`, []string{"echo", "foo", "bar"})
	check(`echo foo   bar`, []string{"echo", "foo", "bar"})
	check("echo foo\tbar", []string{"echo", "foo", "bar"})

	// backslash interpretation
	check(`echo foo\ bar`, []string{"echo", "foo bar"})
	check("echo foo\\\tbar", []string{"echo", "foo\tbar"})
	check("echo foo\\\nbar", []string{"echo", "foobar"})
	check(`echo foo\nbar`, []string{"echo", `foonbar`})

	// single quotes
	check(`echo 'foo bar'`, []string{"echo", "foo bar"})
	check(`echo 'foo\''bar'`, []string{"echo", "foo\\bar"})
	check(`echo 'foo\''bar'`, []string{"echo", "foo\\bar"})
	check(`echo ''`, []string{"echo", ""})
	check(`echo '' ''`, []string{"echo", "", ""})
	check(`echo ''  ''`, []string{"echo", "", ""})

	// double quotes
	check(`echo "foo bar"`, []string{"echo", "foo bar"})
	check(`echo "foo \"bar\""`, []string{"echo", "foo \"bar\""})

	// double quotes
	check(`echo "foo bar"`, []string{"echo", "foo bar"})
	check(`echo "foo \"bar\""`, []string{"echo", "foo \"bar\""})
	check(`echo "foo \$bar"`, []string{"echo", "foo $bar"})
	check(`echo "foo \\bar"`, []string{"echo", `foo \bar`})
	check(`echo "foo \bar"`, []string{"echo", `foo \bar`})
	check(`echo "foo \bar"`, []string{"echo", `foo \bar`})
	check(`echo ""`, []string{"echo", ""})
	check(`echo "" ""`, []string{"echo", "", ""})
	check(`echo ""  ""`, []string{"echo", "", ""})
}
