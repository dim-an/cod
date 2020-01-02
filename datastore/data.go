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
	"cod/util"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrAppPathIsEmpty = fmt.Errorf("executable path cannot be empty")
)

type Policy string

const (
	PolicyUnknown = Policy("")
	PolicyAsk     = Policy("ask")
	PolicyTrust   = Policy("trust")
	PolicyIgnore  = Policy("ignore")
)

type AddHelpPageStatus string

const (
	AddHelpPageStatusNew     = AddHelpPageStatus("new")
	AddHelpPageStatusUpdated = AddHelpPageStatus("updated")
)

type Command struct {
	Args []string
	Env  []string
	Dir  string
}

type Completion struct {
	Flag    string
	Context []string
}

type HelpPage struct {
	ExecutablePath string
	Completions    []Completion
	CheckSum       string
	Command        Command
}

func CheckHelpPage(helpPage *HelpPage) (err error) {
	err = CheckExecutablePath(helpPage.ExecutablePath)
	if err != nil {
		return
	}
	return
}

func CheckExecutablePath(executablePath string) error {
	if len(executablePath) == 0 {
		return ErrAppPathIsEmpty
	}
	if filepath.IsAbs(executablePath) {
		cleaned := filepath.Clean(executablePath)
		if cleaned != executablePath {
			return fmt.Errorf("executable path is not of canonical form: %q", executablePath)
		}
		return nil
	}
	return fmt.Errorf("executable path cannot be relative: %q", executablePath)
}

func CanonizeExecutablePath(name, workDir, pathVar, homeDir string) (canonized string, err error) {
	if len(name) == 0 {
		err = ErrAppPathIsEmpty
		return
	}
	if homeDir != "" && !filepath.IsAbs(homeDir) {
		err = fmt.Errorf("home directory must be absolute: %q", workDir)
		return
	}
	if !filepath.IsAbs(workDir) {
		err = fmt.Errorf("directory must be absolute: %q", workDir)
		return
	}
	if !strings.ContainsRune(name, os.PathSeparator) {
		canonized, err = util.FindExecutable(name, workDir, pathVar)
		return
	}

	if filepath.IsAbs(name) {
		canonized = name
	} else if strings.HasPrefix(name, "~/") {
		if homeDir == "" {
			err = fmt.Errorf("cannot expand ~: home directory is not specified")
			return
		}
		canonized = filepath.Join(homeDir, strings.TrimPrefix(name, "~/"))
	} else {
		canonized = filepath.Join(workDir, name)
	}
	canonized = filepath.Clean(canonized)
	return
}
