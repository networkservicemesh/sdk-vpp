// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package proxy provides method for proxying socket from one net NS to other
package proxy

import (
	"context"
	"net"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type proxyListener struct {
	network string

	targetNSHandle       netns.NsHandle
	targetSocketFilename string

	listener net.Listener

	proxyConnections []*proxyConnection
}

// Start starts proxying {targetNSURL, targetSocketFilename} socket via {nsURL, socketFilename} socket
func Start(
	ctx context.Context, network string,
	nsURL, socketFilename string,
	targetNSURL, targetSocketFilename string,
) (err error) {
	p := &proxyListener{
		network:              network,
		targetNSHandle:       netns.None(),
		targetSocketFilename: targetSocketFilename,
	}
	defer func() {
		if err != nil {
			_ = p.Close()
		}
	}()

	nsHandle, err := nshandle.FromURL(nsURL)
	if err != nil {
		return errors.Wrap(err, "failed to get server net NS handle")
	}
	defer func() { _ = nsHandle.Close() }()

	p.listener, err = listen(p.network, socketFilename, nsHandle)
	if err != nil {
		return errors.Wrap(err, "failed to start listening")
	}

	p.targetNSHandle, err = nshandle.FromURL(targetNSURL)
	if err != nil {
		return errors.Wrap(err, "failed to get server net NS handle")
	}

	// Do a trial dial to ensure we can actually proxy
	trialConn, err := dial(p.network, p.targetSocketFilename, p.targetNSHandle)
	if err != nil {
		return errors.Wrapf(err, "unable to dial %s", targetSocketFilename)
	}
	_ = trialConn.Close()

	go p.accept(ctx)

	return nil
}

func (p *proxyListener) Close() error {
	if p == nil {
		return nil
	}

	var err error
	if p.targetNSHandle.IsOpen() {
		err = multierror.Append(err, p.targetNSHandle.Close())
	}
	if p.listener != nil {
		err = multierror.Append(err, p.listener.Close())
	}
	for _, proxyConn := range p.proxyConnections {
		err = multierror.Append(err, proxyConn.Close())
	}

	return err
}

func (p *proxyListener) accept(ctx context.Context) {
	logger := log.FromContext(ctx).
		WithField("proxy.proxyListener", "accept").
		WithField("proxy", p.listener.Addr().String()).
		WithField("target", p.targetSocketFilename)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		_ = p.listener.Close()
	}()

	defer func() { _ = p.Close() }()
	for {
		in, err := p.listener.Accept()
		if err != nil {
			if optErr, ok := err.(*net.OpError); !ok || !optErr.Temporary() {
				logger.Warnf("failed to accept: %s", err.Error())
				return
			}
		}

		out, err := dial(p.network, p.targetSocketFilename, p.targetNSHandle)
		if err != nil {
			if optErr, ok := err.(*net.OpError); !ok || !optErr.Temporary() {
				logger.Warnf("failed to dial: %s", err.Error())
				_ = in.Close()
				return
			}
		}

		proxyConn, err := newProxyConnection(in, out)
		if err != nil {
			logger.Warnf("failed to copy data: %s", err.Error())
			_ = in.Close()
			_ = out.Close()
			return
		}

		logger.Debug("established a new connection")

		// TODO - clean up - while 99% of the time this won't be an issue because we will have exactly one thing
		//        in this list... in principle it could leak memory
		p.proxyConnections = append(p.proxyConnections, proxyConn)
	}
}
