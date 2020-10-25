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
	"github.com/dim-an/cod/server"
)

type Application interface {
	Config() *server.Configuration
	Client() *server.Client

	Close()
}

func NewApplication() Application {
	return &applicationImpl{}
}

type applicationImpl struct {
	configuration *server.Configuration
	client        *server.Client
}

func (a *applicationImpl) Config() *server.Configuration {
	if a.configuration == nil {
		cfg, err := server.DefaultConfiguration()
		verifyFatal(err)
		a.configuration = &cfg
	}
	return a.configuration
}

func (a *applicationImpl) Client() *server.Client {
	if a.client == nil {
		var err error
		a.client, err = server.NewClient(*a.Config())
		verifyFatal(err)
	}
	return a.client
}

func (a *applicationImpl) Close() {
	if a.client != nil {
		err := a.client.Close()
		verifyFatal(err)
	}
}
