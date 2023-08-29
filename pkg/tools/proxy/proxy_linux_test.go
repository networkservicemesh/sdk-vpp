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

//go:build perm
// +build perm

package proxy_test

import (
	"context"
	"io"
	"net"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/proxy"
)

const (
	network = "unix"
	ping    = "ping"
	pong    = "pong"
)

func TestStartPerm(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	tempDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defaultNS, err := netns.Get()
	require.NoError(t, err)
	defer func() { _ = defaultNS.Close() }()

	proxyNS, err := netns.NewNamed("proxy")
	require.NoError(t, err)
	defer func() { _ = proxyNS.Close() }()

	targetNS, err := netns.NewNamed("target")
	require.NoError(t, err)
	defer func() { _ = targetNS.Close() }()

	// 1. Start listening in target net NS.

	require.NoError(t, netns.Set(targetNS))

	targetFile := "@" + path.Join(tempDir, "target")

	l, err := net.Listen("unix", targetFile)
	require.NoError(t, err)
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	pongCh := doPong(l)

	// 2. Create proxy from default net NS.

	require.NoError(t, netns.Set(defaultNS))

	proxyFile := "@" + path.Join(tempDir, "proxy")

	require.NoError(t, proxy.Start(ctx, network, nsURL("proxy"), proxyFile, nsURL("target"), targetFile))

	// 3. Dial proxy from proxy net NS.

	require.NoError(t, netns.Set(proxyNS))

	conn, err := net.Dial(network, proxyFile)
	require.NoError(t, err)
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	doPing(t, conn)
	require.NoError(t, <-pongCh)
}

func doPong(l net.Listener) <-chan error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		for {
			conn, err := l.Accept()
			if err != nil {
				ch <- err
				return
			}
			defer func() { _ = conn.Close() }()

			buff := make([]byte, 10)
			n, err := conn.Read(buff)
			if err == io.EOF {
				// test dial
				continue
			}
			if err != nil {
				ch <- err
				return
			}
			if msg := string(buff[:n]); msg != ping {
				ch <- errors.Errorf("expected %s, actual %s", ping, msg)
				return
			}

			_, err = conn.Write([]byte(pong))
			if err != nil {
				ch <- err
			}
			return
		}
	}()
	return ch
}

func doPing(t *testing.T, conn io.ReadWriter) {
	_, err := conn.Write([]byte(ping))
	require.NoError(t, err)

	buff := make([]byte, 10)
	n, err := conn.Read(buff)
	if err != nil {
		require.EqualError(t, err, io.EOF.Error())
	}
	require.Equal(t, pong, string(buff[:n]))
}

func nsURL(name string) string {
	return "file://" + path.Join("/run/netns", name)
}
