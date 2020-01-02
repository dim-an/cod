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
	"cod/util"
	"errors"
	"fmt"
)

const (
	GenericError        = 1
	NotImplementedError = 2
	BinaryNotFound      = 3
)

type ErrorResponse struct {
	Code int
	Desc string
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("server returned error: %s", e.Desc)
}

func GetErrorCode(err error) int {
	var response *ErrorResponse
	if errors.As(err, &response) {
		return response.Code
	}
	return GenericError
}

func toErrorResponse(err error) *ErrorResponse {
	if err == nil {
		return nil
	}
	var code int
	switch {
	case errors.Is(err, util.ErrNotImplemented):
		code = NotImplementedError
	case errors.Is(err, util.ErrBinaryNotFound):
		code = BinaryNotFound
	default:
		code = GenericError
	}
	return &ErrorResponse{
		Code: code,
		Desc: err.Error(),
	}
}
