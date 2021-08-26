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
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type statsServer struct {
	chainCtx  context.Context
	statsConn *core.StatsConnection
	once      sync.Once
	initErr   error
}

// NewServer provides a NetworkServiceServer chain elements that retrieves vpp interface metrics.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &statsServer{
		chainCtx: ctx,
	}
}

func (s *statsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	initErr := s.init(s.chainCtx)
	if initErr != nil {
		log.FromContext(ctx).Errorf("%v", initErr)
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil || initErr != nil {
		return conn, err
	}

	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], false)
	return conn, nil
}

func (s *statsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil || s.initErr != nil {
		return rv, err
	}

	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], false)
	return &empty.Empty{}, nil
}

func (s *statsServer) init(chainCtx context.Context) error {
	s.once.Do(func() {
		s.statsConn, s.initErr = initFunc(chainCtx)
	})
	return s.initErr
}
