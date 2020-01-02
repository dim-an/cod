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
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
)

var ErrNotImplemented = fmt.Errorf("not implemented")

func VerifyPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func Purge(dir string) (err error) {
	err = os.RemoveAll(dir)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return
}

func CreateDirIfNotExists(dir string) (err error) {
	stat, err := os.Stat(dir)
	if err == nil {
		if !stat.IsDir() {
			err = fmt.Errorf("%v is not a directory", dir)
			return
		}
	} else if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
	}
	return
}

type Warning struct {
	Warning string
}

type Warner struct {
	Warns []Warning
}

func (w *Warner) WarnError(err error) {
	log.Printf("warn: %v", err)
	if w != nil {
		w.Warns = append(w.Warns, Warning{err.Error()})
	}
}

func (w *Warner) Warnf(format string, v ...interface{}) {
	err := fmt.Errorf(format, v...)
	w.WarnError(err)
}

func LogWarnings(warns []Warning) {
	for _, w := range warns {
		log.Printf("warn: %v", w.Warning)
	}
}

func StringSortUniq(s []string) {
	sort.Strings(s)
	writeIndex := 1
	for i := 1; i < len(s); i += 1 {
		if s[writeIndex-1] == s[i] {
			continue
		}
		if writeIndex != i {
			s[writeIndex] = s[i]
		}
		writeIndex += 1
	}
	return
}

func HashStrings(s []string) string {
	h := sha256.New()

	var res []byte
	for i := range s {
		h.Write(res)
		h.Write([]byte(s[i]))
		res = h.Sum(nil)
		h.Reset()
	}

	return fmt.Sprintf("%x", res)
}
