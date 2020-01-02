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

package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGlobExpression_Match(t *testing.T) {
	match := func(s, glob string) bool {
		g, err := CompileSelector(glob, "/home/user")
		require.NoError(t, err)
		return g.MatchString(s)
	}

	require.True(t, match("/foo/bar", "/foo/bar"))

	require.True(t, match("/foo/bar/baz", "/foo/bar/*"))
	require.True(t, match("/foo/bar/qux", "/foo/bar/*"))
	require.False(t, match("/foo/bar", "/foo/bar/*"))
	require.False(t, match("/foo/bar/qux/baz", "/foo/bar/*"))
	require.False(t, match("/foo/barbaz", "/foo/bar/*"))
	require.False(t, match("/foo/barbaz/qux", "/foo/bar/*"))

	require.True(t, match("/foo/bar/baz", "/foo/bar/**"))
	require.True(t, match("/foo/bar/qux", "/foo/bar/**"))
	require.False(t, match("/foo/bar", "/foo/bar/**"))
	require.True(t, match("/foo/bar/qux/baz", "/foo/bar/**"))
	require.False(t, match("/foo/barbaz", "/foo/bar/*"))
	require.False(t, match("/foo/barbaz/qux", "/foo/bar/*"))

	require.True(t, match("/foo/bar/baz", "**"))
	require.True(t, match("/baz", "**"))

	require.True(t, match("/home/user/my/repo/bin/scripts", "~/my/repo/**"))
	require.False(t, match("/my/repo/script", "~/my/repo/**"))
}
