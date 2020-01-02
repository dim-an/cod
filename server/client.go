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
	"bufio"
	"cod/util"
	"fmt"
	"net"
	"time"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dial(network, address string) (conn net.Conn, err error) {
	for i := 0; i != 20; i += 1 {
		conn, err = net.Dial(network, address)
		if err == nil {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
	return
}

func NewClient(configuration Configuration) (client *Client, err error) {
	conn, err := dial("unix", configuration.GetSocketFile())
	if err != nil {
		return
	}
	client = &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
	return
}

func (c *Client) Request(req interface{}, rsp interface{}) (err error) {
	reqString := MarshalRequest(req)
	reqString = append(reqString, byte('\n'))
	_, err = c.conn.Write(reqString)
	if err != nil {
		return
	}

	data, err := c.reader.ReadBytes('\n')
	if err != nil {
		err = fmt.Errorf("cannot read server response: %w", err)
		return
	}
	err, warns := UnmarshalResponseToVar(data, rsp)
	util.LogWarnings(warns)
	return
}

func (c *Client) Close() (err error) {
	err = c.conn.Close()
	return
}
