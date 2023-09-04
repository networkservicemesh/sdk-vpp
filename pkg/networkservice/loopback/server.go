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

package loopback

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type loopbackServer struct {
	vppConn api.Connection

	loopbacks *Map
}

// NewServer creates a NetworkServiceServer chain element to create the loopback vpp-interface
func NewServer(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceServer {
	o := &options{
		loopbacks: NewMap(),
	}
	for _, opt := range opts {
		opt(o)
	}

	return &loopbackServer{
		vppConn:   vppConn,
		loopbacks: o.loopbacks,
	}
}

func (l *loopbackServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	networkService := request.GetConnection().NetworkService
	if err := createLoopback(ctx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l)); err != nil {
		return nil, err
	}
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		del(closeCtx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l))
	}
	return conn, err
}

func (l *loopbackServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	del(ctx, l.vppConn, conn.NetworkService, l.loopbacks, metadata.IsClient(l))
	return next.Server(ctx).Close(ctx, conn)
}
