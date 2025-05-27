// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2025 Nordix Foundation.
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

package stats

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"go.fd.io/govpp/core"
)

type statsServer struct {
	chainCtx  context.Context
	statsConn *core.StatsConnection
	statsSock string
	once      sync.Once
	initErr   error
}

// NewServer provides a NetworkServiceServer chain elements that retrieves vpp interface statistics.
func NewServer(ctx context.Context, options ...Option) networkservice.NetworkServiceServer {
	prometheusInitOnce.Do(registerMetrics)
	opts := &statsOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return &statsServer{
		chainCtx:  ctx,
		statsSock: opts.socket,
	}
}

func (s *statsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	initErr := s.init()
	if initErr != nil {
		log.FromContext(ctx).Errorf("%v", initErr)
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil || initErr != nil {
		return conn, err
	}

	nscInterface := conn.Path.PathSegments[0].Metrics["client_interface"]
	nseInterface := conn.Path.PathSegments[len(conn.Path.PathSegments)-1].Metrics["server_interface"]
	if len(conn.Path.PathSegments) > 4 {
		if conn.Path.Index < 4 {
			nseInterface = ""
		} else {
			nscInterface = ""
		}
	}

	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], false,
		conn.Id, conn.NetworkService, conn.Path.PathSegments[0].Id, nscInterface, nseInterface, false)
	return conn, nil
}

func (s *statsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil || s.initErr != nil {
		return rv, err
	}

	nscInterface := conn.Path.PathSegments[0].Metrics["client_interface"]
	nseInterface := conn.Path.PathSegments[len(conn.Path.PathSegments)-1].Metrics["server_interface"]
	if len(conn.Path.PathSegments) > 4 {
		if conn.Path.Index < 4 {
			nseInterface = ""
		} else {
			nscInterface = ""
		}
	}

	retrieveMetrics(ctx, s.statsConn, conn.Path.PathSegments[conn.Path.Index], false,
		conn.Id, conn.NetworkService, conn.Path.PathSegments[0].Id, nscInterface, nseInterface, true)
	return &empty.Empty{}, nil
}

func (s *statsServer) init() error {
	s.once.Do(func() {
		s.statsConn, s.initErr = initFunc(s.chainCtx, s.statsSock)
	})
	return s.initErr
}
