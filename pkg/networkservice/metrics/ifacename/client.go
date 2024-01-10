// Copyright (c) 2024 Cisco and/or its affiliates.
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

package ifacename

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/core"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ifaceNamesClient struct {
	chainCtx  context.Context
	statsConn *core.StatsConnection
	vppConn   api.Connection
	statsSock string
	once      sync.Once
	initErr   error
}

// NewClient provides a NetworkServiceClient chain elements that retrieves vpp interface metrics.
func NewClient(ctx context.Context, vppConn api.Connection, options ...Option) networkservice.NetworkServiceClient {
	opts := &ifacenameOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return &ifaceNamesClient{
		chainCtx:  ctx,
		vppConn:   vppConn,
		statsSock: opts.Socket,
	}
}

func (s *ifaceNamesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	initErr := s.init()
	if initErr != nil {
		log.FromContext(ctx).Errorf("%v", initErr)
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil || initErr != nil {
		return conn, err
	}

	retrieveIfaceNames(ctx, s.statsConn, s.vppConn, conn, true)

	return conn, nil
}

func (s *ifaceNamesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil || s.initErr != nil {
		return rv, err
	}

	retrieveIfaceNames(ctx, s.statsConn, s.vppConn, conn, true)

	return &empty.Empty{}, nil
}

func (s *ifaceNamesClient) init() error {
	s.once.Do(func() {
		s.statsConn, s.initErr = initFunc(s.chainCtx, s.statsSock)
	})
	return s.initErr
}
