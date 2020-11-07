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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/dim-an/cod/datastore"
	"github.com/dim-an/cod/util"
)

type AttachRequest struct {
	Shell         string
	Pid           int
	CodBinaryPath string
}

type AttachResponse struct {
}

type CompleteWordsRequest struct {
	// First word of the `Words` must be executable path.
	Words []string
	CWord int
}

type CompleteWordsResponse struct {
	Completions []string
}

type DetachRequest struct {
	Pid int
}

type DetachResponse struct {
}

type InitScriptRequest struct {
	Pid int
}

type InitScriptResponse struct {
	Script []string
}

type ShellAndPid struct {
	Shell string
	Pid   int
}

type ListCommandsRequest struct {
	Selectors []string
}

type ListCommandsResponseItem struct {
	Id int64

	// In rare cases Command might be empty.
	Command *datastore.Command
}
type ListCommandsResponse struct {
	CommandItems []ListCommandsResponseItem
}

type ListClientsRequest struct {
}

type ListClientsResponse struct {
	Clients []ShellAndPid
}

type RemoveCommandsRequest struct {
	HelpPageIds []int64
}

type RemoveCommandsResponse struct {
}

type AddHelpPageRequest struct {
	Command datastore.Command
	Policy  datastore.Policy
}

type AddHelpPageResponse struct {
	HelpPage datastore.HelpPage
	Status   datastore.AddHelpPageStatus
}

type ParseCommandLineRequest struct {
	Pid         int
	CommandLine string
	Dir         string
	Env         []string
}

type ParseCommandLineResponse struct {
	IsHelpCommand bool
	PolicyMode    datastore.Policy
	Args          []string
	Env           []string
}

type PollUpdatesRequest struct {
	Pid int
}

type PollUpdatesResponse struct {
	Script []string
}

type UpdateHelpPageRequest struct {
	Id      int64
	Command datastore.Command
}

type UpdateHelpPageResponse struct {
}

type RemoteError struct {
	Code    int
	Message string
}

func (e RemoteError) Error() string {
	return fmt.Sprintf("server returned error: %s", e.Message)
}

func verifyMessageType(msg interface{}) {
	if msg != nil {
		isRequest(msg)
	}
}

func verifyResponseType(msg interface{}) {
	if msg != nil && isRequest(msg) {
		panic("expected response type")
	}
}

func verifyRequestType(msg interface{}) {
	if !isRequest(msg) {
		panic("expected request type")
	}
}

func getMessageName(msg interface{}) string {
	verifyMessageType(msg)
	return reflect.TypeOf(msg).Elem().Name()
}

func isRequest(msg interface{}) bool {
	switch msg.(type) {
	case *AttachRequest,
		*CompleteWordsRequest,
		*DetachRequest,
		*InitScriptRequest,
		*ListClientsRequest,
		*ListCommandsRequest,
		*RemoveCommandsRequest,
		*AddHelpPageRequest,
		*ParseCommandLineRequest,
		*PollUpdatesRequest,
		*UpdateHelpPageRequest:
		return true
	case *AttachResponse,
		*CompleteWordsResponse,
		*DetachResponse,
		*InitScriptResponse,
		*ListClientsResponse,
		*ListCommandsResponse,
		*RemoveCommandsResponse,
		*AddHelpPageResponse,
		*ParseCommandLineResponse,
		*PollUpdatesResponse,
		*UpdateHelpPageResponse:
		return false
	default:
		panic(fmt.Errorf("unexpected type: %v", reflect.TypeOf(msg)))
	}
}

type requestOnWire struct {
	Request string
	Payload interface{}
}

type responseOnWire struct {
	Response interface{}
	Error    *ErrorResponse `json:",omitempty"`
	Warnings []util.Warning `json:",omitempty"`
}

func MarshalRequest(req interface{}) (bytes []byte) {
	verifyRequestType(req)

	wire := requestOnWire{
		Request: getMessageName(req),
		Payload: req,
	}

	bytes, err := json.Marshal(&wire)
	if err != nil {
		panic(err)
	}
	return
}

func UnmarshalRequest(data []byte) (name string, payload interface{}, err error) {
	wire := requestOnWire{
		Payload: &payload,
	}
	err = json.Unmarshal(data, &wire)
	if err != nil {
		return
	}
	name = wire.Request
	return
}

func CastRequestPayload(payload interface{}, req interface{}) {
	bytes, err := json.Marshal(payload)
	util.VerifyPanic(err)
	err = json.Unmarshal(bytes, req)
	util.VerifyPanic(err)
}

func MarshalResponse(rsp interface{}, e error, warns []util.Warning) (bytes []byte) {
	verifyResponseType(rsp)

	wire := responseOnWire{
		Response: rsp,
		Error:    toErrorResponse(e),
		Warnings: warns,
	}
	bytes, err := json.Marshal(&wire)
	util.VerifyPanic(err)
	return
}

func UnmarshalResponseToVar(data []byte, rsp interface{}) (err error, warns []util.Warning) {
	verifyResponseType(rsp)

	wire := responseOnWire{
		Response: rsp,
	}

	err = json.Unmarshal(data, &wire)
	if err != nil {
		return
	}

	if wire.Error != nil {
		err = wire.Error
		return
	}
	warns = wire.Warnings
	return
}
