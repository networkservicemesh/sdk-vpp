// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package memifproxy

import (
	"context"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/proxy"
)

const memifNetwork = "unixpacket"

type memifProxyServer struct {
	chainCtx context.Context
}

// NewServer - create a new memifProxy server chain element
func NewServer(chainCtx context.Context) networkservice.NetworkServiceServer {
	return &memifProxyServer{
		chainCtx: chainCtx,
	}
}

func (m *memifProxyServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	var mechanism *memifMech.Mechanism
	if mechanism = memifMech.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		storeInfo(ctx, new(Info))
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// If it is NOT a direct memif case, do nothing.
	info, _ := LoadInfo(ctx)
	if info.SocketFile == "" {
		return conn, nil
	}

	if err = create(ctx, m.chainCtx, conn, info); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := m.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func create(ctx, chainCtx context.Context, conn *networkservice.Connection, info *Info) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// This connection has already been created
		if _, ok := load(ctx); ok {
			return nil
		}

		proxyCtx, cancelProxy := context.WithCancel(chainCtx)

		if err := proxy.Start(
			proxyCtx, memifNetwork,
			mechanism.GetNetNSURL(), listenSocketFilename(conn),
			info.NSURL, info.SocketFile,
		); err != nil {
			cancelProxy()
			return err
		}
		store(ctx, cancelProxy)

		mechanism.SetSocketFilename(listenSocketFilename(conn))
	}
	return nil
}

func (m *memifProxyServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if cancelProxy, ok := loadAndDelete(ctx); ok {
			cancelProxy()
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}

func listenSocketFilename(conn *networkservice.Connection) string {
	return "@" + filepath.Join(os.TempDir(), "memifproxy", conn.GetId(), "memif.socket")
}
