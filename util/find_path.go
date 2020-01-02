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

var (
	ErrBinaryNotFound = fmt.Errorf("cannot find binary in PATH")
)

func checkExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}

func FindExecutable(name, dir, pathVar string) (string, error) {
	if strings.ContainsRune(name, os.PathSeparator) {
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		return path, nil
	}
	for _, pathDir := range filepath.SplitList(pathVar) {
		if pathDir == "" {
			pathDir = "."
		}
		path := filepath.Join(pathDir, name)
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		err := checkExecutable(path)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, name)
}

func GetPathVar(environ []string) string {
	for _, e := range environ {
		if strings.HasPrefix(e, "PATH=") {
			return strings.TrimPrefix(e, "PATH=")
		}
	}
	return ""
}

func GetHomeVar(environ []string) string {
	for _, e := range environ {
		if strings.HasPrefix(e, "HOME=") {
			return strings.TrimPrefix(e, "HOME=")
		}
	}
	return ""
}
