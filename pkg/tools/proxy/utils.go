// Copyright (c) 2022 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package proxy

import (
	"net"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

func listen(network, address string, nsHandle netns.NsHandle) (ln net.Listener, err error) {
	current, err := nshandle.Current()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current net NS")
	}
	defer func() { _ = current.Close() }()

	err = nshandle.RunIn(current, nsHandle, func() error {
		var listenErr error
		ln, listenErr = net.Listen(network, address)
		return listenErr
	})

	return ln, err
}

func dial(network, address string, nsHandle netns.NsHandle) (conn net.Conn, err error) {
	current, err := nshandle.Current()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current net NS")
	}
	defer func() { _ = current.Close() }()

	err = nshandle.RunIn(current, nsHandle, func() error {
		var dialErr error
		conn, dialErr = net.Dial(network, address)
		return dialErr
	})

	return conn, err
}
