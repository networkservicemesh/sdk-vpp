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
	"context"
	"net/url"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

const (
	memifNetwork = "unixpacket"
	maxFDCount   = 1
	bufferSize   = 128
)

type memifProxyClient struct{}

// New - create a new memifProxy client chain element
func New() networkservice.NetworkServiceClient {
	return &memifProxyClient{}
}

func (m *memifProxyClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	if mechanism := memifMech.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		if listener, ok := load(ctx, metadata.IsClient(m)); ok {
			mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: listener.socketFilename}).String())
		}
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	mechanism := memifMech.ToMechanism(conn.GetMechanism())
	if mechanism == nil {
		return conn, nil
	}

	// If we are already running a proxy... just keep running it
	if _, ok := load(ctx, true); ok {
		mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: listenSocketFilename(conn)}).String())
		return conn, nil
	}

	if err = os.MkdirAll(filepath.Dir(listenSocketFilename(conn)), 0700); err != nil {
		err = errors.Wrapf(err, "unable to mkdir %s", filepath.Dir(listenSocketFilename(conn)))
		if closeErr := m.closeOnFailure(postponeCtxFunc, conn, opts); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		return nil, err
	}

	listener, err := newProxyListener(mechanism, listenSocketFilename(conn))
	if err != nil {
		if closeErr := m.closeOnFailure(postponeCtxFunc, conn, opts); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		return nil, err
	}

	store(ctx, metadata.IsClient(m), listener)

	return conn, nil
}

func (m *memifProxyClient) closeOnFailure(postponeCtxFunc func() (context.Context, context.CancelFunc), conn *networkservice.Connection, opts []grpc.CallOption) error {
	closeCtx, cancelClose := postponeCtxFunc()
	defer cancelClose()

	_, err := m.Close(closeCtx, conn, opts...)

	return err
}

func (m *memifProxyClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if listener, ok := load(ctx, metadata.IsClient(m)); ok {
			mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: listener.socketFilename}).String())
		}
	}

	rv, err := next.Client(ctx).Close(ctx, conn)
	if listener, ok := loadAndDelete(ctx, metadata.IsClient(m)); ok {
		_ = listener.Close()
	}
	_ = os.RemoveAll(filepath.Dir(listenSocketFilename(conn)))
	return rv, err
}

func listenSocketFilename(conn *networkservice.Connection) string {
	return filepath.Join(os.TempDir(), "memifproxy", conn.GetId(), "memif.socket")
}
