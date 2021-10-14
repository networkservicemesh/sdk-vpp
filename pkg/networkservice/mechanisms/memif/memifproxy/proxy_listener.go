// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

// +build !windows

package memifproxy

import (
	"net"
	"net/url"

	"github.com/hashicorp/go-multierror"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/pkg/errors"
)

type proxyListener struct {
	listener         net.Listener
	socketFilename   string
	proxyConnections []*proxyConnection
}

func newProxyListener(mechanism *memifMech.Mechanism, listenSocketFilename string) (*proxyListener, error) {
	// Extract the socket filename
	u, err := url.Parse(mechanism.GetSocketFileURL())
	if err != nil {
		return nil, errors.Wrapf(err, "not a valid url %q", mechanism.GetSocketFileURL())
	}
	if u.Scheme != memifMech.SocketFileScheme {
		return nil, errors.Errorf("socket file url must have scheme %q, actual %q", memifMech.SocketFileScheme, u.Scheme)
	}
	p := &proxyListener{
		socketFilename: u.Path,
	}

	// Do a trial dial to ensure we can actually proxy
	trialConn, err := net.Dial(memifNetwork, u.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "proxyListener unable to dial %s", p.socketFilename)
	}
	_ = trialConn.Close()

	p.listener, err = net.Listen(memifNetwork, listenSocketFilename)
	if err != nil {
		return nil, errors.Wrapf(err, "proxyListener unable to listen on %s", listenSocketFilename)
	}
	go p.accept()
	mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: listenSocketFilename}).String())
	return p, nil
}

func (p *proxyListener) accept() {
	defer func() { _ = p.Close() }()
	for {
		in, err := p.listener.Accept()
		if err != nil {
			if optErr, ok := err.(*net.OpError); !ok || !optErr.Temporary() {
				// TODO - perhaps log this?
				return
			}
		}

		out, err := net.Dial(memifNetwork, p.socketFilename)
		if err != nil {
			if optErr, ok := err.(*net.OpError); !ok || !optErr.Temporary() {
				_ = in.Close()
				// TODO - perhaps log this?
				return
			}
		}

		proxyConn, err := newProxyConnection(in, out)
		if err != nil {
			_ = in.Close()
			_ = out.Close()
			// TODO - perhaps log this?
			return
		}

		// TODO - clean up - while 99% of the time this won't be an issue because we will have exactly one thing
		//        in this list... in principle it could leak memory
		p.proxyConnections = append(p.proxyConnections, proxyConn)
	}
}

func (p *proxyListener) Close() error {
	if p == nil {
		return nil
	}
	err := p.listener.Close()
	for _, proxyConn := range p.proxyConnections {
		err = multierror.Append(err, proxyConn.Close())
	}
	return err
}
