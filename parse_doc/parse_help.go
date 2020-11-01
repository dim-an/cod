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

package parse_doc

import (
	"crypto/sha1"
	"fmt"
	"log"

	"github.com/dim-an/cod/datastore"
)

type HelpParser interface {
	Name() string
	Parse(context parseContext) (*parseResult, error)
}

var parsers = []HelpParser{
	makeArgparseParser(),
	makeDefaultParser(),
}

func ParseHelp(args []string, help string) (*datastore.HelpPage, error) {
	if len(args) < 1 {
		log.Panicf("args cannot be empty")
	}
	preparedText, err := makePreparedText(help)
	if err != nil {
		return nil, err
	}

	ctx := parseContext{
		args: args,
		text: preparedText,
	}

	var res *parseResult
	for idx := range parsers {
		var err error
		res, err = parsers[idx].Parse(ctx)
		if err != nil {
			log.Printf("Parser %s failed with error %s", parsers[idx].Name(), err)
			continue
		}
		break
	}
	if res == nil {
		panic("expected default parser to parse help successfully")
	}

	helpPage := datastore.HelpPage{
		ExecutablePath: args[0],
		Completions:    res.completions,
	}
	helpPage.CheckSum = fmt.Sprintf("%x", sha1.Sum([]byte(help)))
	return &helpPage, nil
}
