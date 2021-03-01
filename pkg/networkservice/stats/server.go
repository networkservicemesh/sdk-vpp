// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"strconv"
	"sync"

	"git.fd.io/govpp.git/adapter"
	"git.fd.io/govpp.git/adapter/statsclient"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type statsServer struct {
	statsConn *core.StatsConnection

	once    sync.Once
	initErr error
}

// NewServer provides a NetworkServiceServer chain elements that retrieves vpp interface metrics.
func NewServer() networkservice.NetworkServiceServer {
	return &statsServer{}
}

func (s *statsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	errInit := s.init()
	if errInit != nil {
		log.FromContext(ctx).Errorf("%v", errInit)
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil || errInit != nil {
		return conn, err
	}

	s.retrieveMetrics(ctx, conn.Path.PathSegments[conn.Path.Index])
	return conn, nil
}

func (s *statsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}

	s.retrieveMetrics(ctx, conn.Path.PathSegments[conn.Path.Index])
	s.statsConn.Disconnect()
	return &empty.Empty{}, nil
}

func (s *statsServer) init() error {
	s.once.Do(func() {
		s.statsConn, s.initErr = core.ConnectStats(statsclient.NewStatsClient(adapter.DefaultStatsSocket))
	})
	return s.initErr
}

// Save retrieved vpp interface metrics in pathSegment
func (s *statsServer) retrieveMetrics(ctx context.Context, segment *networkservice.PathSegment) {
	stats := new(api.InterfaceStats)
	if e := s.statsConn.GetInterfaceStats(stats); e != nil {
		log.FromContext(ctx).Errorf("getting interface stats failed:", e)
		return
	}

	for idx := range stats.Interfaces {
		iface := &stats.Interfaces[idx]
		if segment.Metrics == nil {
			segment.Metrics = make(map[string]string)
		}
		segment.Metrics[iface.InterfaceName+"_rx_bytes"] = strconv.FormatUint(iface.Rx.Bytes, 10)
		segment.Metrics[iface.InterfaceName+"_tx_bytes"] = strconv.FormatUint(iface.Tx.Bytes, 10)
		segment.Metrics[iface.InterfaceName+"_rx_packets"] = strconv.FormatUint(iface.Rx.Packets, 10)
		segment.Metrics[iface.InterfaceName+"_tx_packets"] = strconv.FormatUint(iface.Tx.Packets, 10)
		segment.Metrics[iface.InterfaceName+"_drops"] = strconv.FormatUint(iface.Drops, 10)
	}
}
