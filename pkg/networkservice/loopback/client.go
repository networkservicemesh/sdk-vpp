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

package loopback

import (
	"context"

	"google.golang.org/grpc"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type loopbackClient struct {
	vppConn api.Connection

	loopbacks *LoopMap
}

// NewClient creates a NetworkServiceClient chain element to create the loopback vpp-interface
func NewClient(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		loopbacks: CreateLoopbackMap(),
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
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		if e := del(ctx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l)); e != nil {
			log.FromContext(ctx).Errorf("unable to delete loopback interface: %v", e)
		}
	}
	return conn, err
}

func (l *loopbackClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if err := del(ctx, l.vppConn, conn.NetworkService, l.loopbacks, metadata.IsClient(l)); err != nil {
		log.FromContext(ctx).Errorf("unable to delete loopback interface: %v", err)
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
