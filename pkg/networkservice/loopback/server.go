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

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
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
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if e := del(ctx, l.vppConn, networkService, l.loopbacks, metadata.IsClient(l)); e != nil {
			log.FromContext(ctx).Errorf("unable to delete loopback interface: %v", e)
		}
	}
	return conn, err
}

func (l *loopbackServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if err := del(ctx, l.vppConn, conn.NetworkService, l.loopbacks, metadata.IsClient(l)); err != nil {
		log.FromContext(ctx).Errorf("unable to delete loopback interface: %v", err)
	}
	return next.Server(ctx).Close(ctx, conn)
}
