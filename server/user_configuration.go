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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/dim-an/cod/datastore"
	"github.com/dim-an/cod/util"
	"github.com/pelletier/go-toml"
)

type Rule struct {
	Executable   string `toml:"executable"`
	compiledGlob util.Selector
	Policy       datastore.Policy `toml:"policy"`
}

type UserConfiguration struct {
	Rules                   []Rule `toml:"rule"`
	commandExecutionTimeout int    `toml:"command-execution-timeout"`
	// NOTE: defaults are set inside LoadUserConfigurationFromBytes
}

func (cfg *UserConfiguration) GetCommandExecutionTimeout() time.Duration {
	return time.Millisecond * time.Duration(cfg.commandExecutionTimeout)
}

func initRule(rule *Rule, homeDir string) (err error) {
	switch rule.Policy {
	case datastore.PolicyAsk, datastore.PolicyIgnore, datastore.PolicyTrust:
		// Policy is ok, do nothing
	default:
		return fmt.Errorf("bad policy: %v", rule.Policy)
	}

	if len(rule.Executable) == 0 {
		return fmt.Errorf(`found rule with empty "executable"`)
	}

	rule.compiledGlob, err = util.CompileSelector(rule.Executable, homeDir)
	if err != nil {
		return fmt.Errorf("bad glob in configuration: %q: %w", rule.Executable, err)
	}
	return nil
}

func LoadUserConfiguration(filename, homeDir string) (userConfiguration UserConfiguration, err error) {
	var bytes []byte
	bytes, err = ioutil.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
			bytes = nil
			// NB. we want to call LoadUserConfigurationFromBytes because it
			// sets defaults
		} else {
			return
		}
	}
	userConfiguration, err = LoadUserConfigurationFromBytes(bytes, homeDir)
	if err != nil {
		err = fmt.Errorf("error parsing %q: %w", filename, err)
	}
	return
}

func LoadUserConfigurationFromBytes(bytes []byte, homeDir string) (userConfiguration UserConfiguration, err error) {
	userConfiguration.commandExecutionTimeout = 1000

	err = toml.Unmarshal(bytes, &userConfiguration)
	if err != nil {
		return
	}
	for i := range userConfiguration.Rules {
		err = initRule(&userConfiguration.Rules[i], homeDir)
		if err != nil {
			return
		}
	}
	if userConfiguration.commandExecutionTimeout < 0 {
		err = fmt.Errorf("'command-execution-timeout' must not be negative")
		return
	}
	return
}

func (cfg *UserConfiguration) GetExecutablePolicy(executablePath string) datastore.Policy {
	for _, rule := range cfg.Rules {
		ok := rule.compiledGlob.MatchString(executablePath)
		if ok {
			return rule.Policy
		}
	}
	return datastore.PolicyUnknown
}
