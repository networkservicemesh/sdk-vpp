// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package memifrxmode

import (
	"context"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type memifrxmodeServer struct {
	chainCtx context.Context
	vppConn  Connection
}

// NewServer - create a new memifProxy server chain element
func NewServer(chainCtx context.Context, vppConn Connection) networkservice.NetworkServiceServer {
	return &memifrxmodeServer{
		chainCtx: chainCtx,
		vppConn:  vppConn,
	}
}

func (m *memifrxmodeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism == nil {
		return conn, errors.Wrap(err, "failed to get memif mechanism")
	}

	if ok := load(ctx, metadata.IsClient(m)); !ok {
		swIfIndex, _ := ifindex.Load(ctx, metadata.IsClient(m))

		cancelCtx, cancel := context.WithCancel(m.chainCtx)
		store(ctx, metadata.IsClient(m), cancel)

		if err := setRxMode(cancelCtx, m.vppConn, swIfIndex); err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := m.Close(closeCtx, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}

			return nil, err
		}
	}

	return conn, nil
}

func (m *memifrxmodeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		oldCancel()
	}
	return next.Server(ctx).Close(ctx, conn)
}
