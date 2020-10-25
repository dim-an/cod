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

package datastore

import (
	"github.com/dim-an/cod/util"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCanonizeExecutablePath(t *testing.T) {
	var (
		canonized string
		err       error
	)

	canonized, err = CanonizeExecutablePath("foo", "/a/b", "", "")
	require.Error(t, util.ErrBinaryNotFound, err)

	canonized, err = CanonizeExecutablePath("./foo", "/a/b", "", "")
	require.Nil(t, err)
	require.Equal(t, "/a/b/foo", canonized)

	canonized, err = CanonizeExecutablePath("../foo", "/a/b", "", "")
	require.Nil(t, err)
	require.Equal(t, "/a/foo", canonized)

	canonized, err = CanonizeExecutablePath("/foo", "/a/b", "", "")
	require.Nil(t, err)
	require.Equal(t, "/foo", canonized)

	canonized, err = CanonizeExecutablePath("~/foo", "/a/b", "", "")
	require.Error(t, err)

	canonized, err = CanonizeExecutablePath("~/foo", "/a/b", "", "/home/user")
	require.NoError(t, err)
	require.Equal(t, "/home/user/foo", canonized)
}

func TestIsCommandMatchingContext(t *testing.T) {
	require.Equal(t, true,
		IsCommandMatchingContext([]string{"foo", "bar"}, FlagContext{}),
	)
	require.Equal(t, true,
		IsCommandMatchingContext([]string{"foo", "bar"}, FlagContext{SubCommand: []string{"bar"}}),
	)
	require.Equal(t, false,
		IsCommandMatchingContext([]string{"foo", "bar"}, FlagContext{SubCommand: []string{"bar", "baz"}}),
	)
}
