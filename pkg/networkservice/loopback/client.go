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

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type loopbackClient struct {
	vppConn api.Connection

	loopbacks *Map
}

// NewClient creates a NetworkServiceClient chain element to create the loopback vpp-interface
func NewClient(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		loopbacks: NewMap(),
	}
	for _, opt := range opts {
		opt(o)
	}

	return &loopbackClient{
		vppConn:   vppConn,
		loopbacks: o.loopbacks,
	}
}

func (l *loopbackClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	networkService := request.GetConnection().NetworkService
	if err := createLoopback(ctx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l)); err != nil {
		return nil, err
	}
	postponeCtxFunc := postpone.ContextWithValues(ctx)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		del(closeCtx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l))
	}
	return conn, err
}

func (l *loopbackClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	del(ctx, l.vppConn, conn.NetworkService, l.loopbacks, metadata.IsClient(l))
	return next.Client(ctx).Close(ctx, conn, opts...)
}
