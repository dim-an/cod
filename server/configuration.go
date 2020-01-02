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

package server

import (
	"fmt"
	"os"
	"os/user"
	"path"
)

type Configuration struct {
	// it's not real app name but either 'cod' or 'cod-test'
	appName   string
	configDir string
	dataDir   string
	runDir    string
	homeDir   string
}

func DefaultConfiguration() (cfg Configuration, err error) {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	appName := path.Base(executable)
	if appName != "cod" {
		appName = "cod-test"
	}

	usr, err := user.Current()
	if err != nil {
		return
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if len(configDir) == 0 {
		configDir = path.Join(usr.HomeDir, ".config")
	}
	configDir = path.Join(configDir, appName)

	dataDir := os.Getenv("XDG_DATA_HOME")
	if len(dataDir) == 0 {
		dataDir = path.Join(usr.HomeDir, ".local", "share")
	}
	dataDir = path.Join(dataDir, appName)

	runDir := path.Join(dataDir, "var")

	homeDir := os.Getenv("HOME")

	cfg = Configuration{
		appName:   appName,
		configDir: configDir,
		dataDir:   dataDir,
		runDir:    runDir,
		homeDir:   homeDir,
	}

	if len(cfg.GetSocketFile()) > 100 {
		err = fmt.Errorf("socket name %s is too long", cfg.GetSocketFile())
		return
	}
	return
}

func (cfg *Configuration) GetHomeDir() string {
	return cfg.homeDir
}

func (cfg *Configuration) GetRunDir() string {
	return cfg.runDir
}

func (cfg *Configuration) GetCompletionConfigDir() string {
	return path.Join(cfg.configDir, "completions")
}

func (cfg *Configuration) GetUserConfiguration() string {
	return path.Join(cfg.configDir, "config.toml")
}

func (cfg *Configuration) GetCompletionDbDir() string {
	return path.Join(cfg.dataDir, "completions")
}

func (cfg *Configuration) GetCompletionsSqliteDb() string {
	return path.Join(cfg.dataDir, "db.sqlite3")
}

func (cfg *Configuration) GetSocketFile() string {
	return path.Join(cfg.runDir, cfg.appName+".sock")
}

func (cfg *Configuration) GetLockFile() string {
	return path.Join(cfg.runDir, cfg.appName+".lock")
}

func (cfg *Configuration) GetLogDir() string {
	return path.Join(cfg.dataDir, "log")
}

func (cfg *Configuration) GetPidFile() string {
	return path.Join(cfg.runDir, cfg.appName+".pid")
}

func (cfg *Configuration) GetLearnBlacklistFile() string {
	return path.Join(cfg.dataDir, "learn-blacklist.txt")
}

func (cfg *Configuration) GetKnownCommandsFile() string {
	return path.Join(cfg.dataDir, "known-commands.toml")
}
