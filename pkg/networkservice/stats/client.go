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

// +build linux

package stats

import (
	"context"
	"sync"

	"git.fd.io/govpp.git/core"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type statsClient struct {
	chainCtx  context.Context
	statsConn *core.StatsConnection
	once      sync.Once
	initErr   error
}

// NewClient provides a NetworkServiceClient chain elements that retrieves vpp interface metrics.
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	return &statsClient{
		chainCtx: ctx,
	}
}

func (s *statsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	initErr := s.init(s.chainCtx)
	if initErr != nil {
		log.FromContext(ctx).Errorf("%v", initErr)
	}
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil || initErr != nil {
		return nil, err
	}
	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], true)
	return conn, nil
}

func (s *statsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil || s.initErr != nil {
		return rv, err
	}

	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], true)
	return &empty.Empty{}, nil
}

func (s *statsClient) init(chainCtx context.Context) error {
	s.once.Do(func() {
		s.statsConn, s.initErr = initFunc(chainCtx)
	})
	return s.initErr
}
