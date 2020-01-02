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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Selector interface {
	MatchString(s string) bool
}

func CompileSelector(glob, homeDir string) (g Selector, err error) {
	if strings.HasPrefix(glob, "~/") {
		if len(homeDir) == 0 {
			err = fmt.Errorf("cannot expand %q pattern, home directory is unknown", homeDir)
			return
		}
		glob = filepath.Join(homeDir, strings.TrimPrefix(glob, "~/"))
	}

	dir, name := filepath.Split(glob)

	check := func(s string) error {
		if strings.IndexRune(s, '*') != -1 {
			return fmt.Errorf("bad glob: '*' and '**' are supported only as last component of the path")
		}
		return nil
	}

	if name == "*" {
		err = check(dir)
		if err != nil {
			return
		}
		g = starGlob(dir)
	} else if name == "**" {
		err = check(dir)
		if err != nil {
			return
		}
		g = starStarGlob(dir)
	} else if strings.ContainsRune(glob, os.PathSeparator) {
		err = check(glob)
		if err != nil {
			return
		}
		g = noStarGlob(glob)
	} else {
		err = check(glob)
		if err != nil {
			return
		}
		g = baseNameGlob(glob)
	}
	return
}

type baseNameGlob string

func (g baseNameGlob) MatchString(s string) bool {
	return string(g) == filepath.Base(s)
}

type noStarGlob string

func (g noStarGlob) MatchString(s string) bool {
	return string(g) == s
}

type starGlob string

func (g starGlob) MatchString(s string) bool {
	dir, _ := filepath.Split(s)
	return string(g) == dir
}

type starStarGlob string

func (g starStarGlob) MatchString(s string) bool {
	return strings.HasPrefix(s, string(g))
}
