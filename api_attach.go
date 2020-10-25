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

package main

import (
	"fmt"
	"github.com/dim-an/cod/server"
	"time"
)

func getLogFileBaseName(t time.Time) string {
	return fmt.Sprintf("cod.%v.log", t.Format("2006-01-02"))
}

func shellApiAttachMain(pid uint, shell string) {
	config, err := server.DefaultConfiguration()
	if err != nil {
		fatal(err)
	}

	// Trying to daemonize.
	err = daemonize(&config)
	verifyFatal(err)

	client, err := server.NewClient(config)
	if err != nil {
		fatal(err)
	}

	rsp := server.AttachResponse{}
	req := server.AttachRequest{
		Pid:   int(pid),
		Shell: shell,
	}

	err = client.Request(&req, &rsp)
	if err != nil {
		fatal(err)
	}
}
