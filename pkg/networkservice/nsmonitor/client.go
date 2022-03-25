// Copyright (c) 2022 Cisco and/or its affiliates.
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

package nsmonitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type netNSMonitorClient struct {
	monitor *netNSMonitor
}

// NewClient returns new net ns monitoring client
func NewClient() networkservice.NetworkServiceClient {
	return &netNSMonitorClient{
		monitor: newMonitor(),
	}
}

func (r *netNSMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	inodeURL, ok := conn.GetMechanism().GetParameters()[common.InodeURL]
	logger := log.FromContext(ctx).WithField("component", "netNsMonitor").WithField(common.InodeURL, inodeURL)
	if ok {
		result, err := r.monitor.AddNSInode(ctx, inodeURL)
		if err != nil {
			logger.WithField("error", err).Error("unable to monitor")
			return nil, err
		}
		logger.Info(result)
	} else {
		logger.Info("inodeURL not found")
	}

	return conn, nil
}

func (r *netNSMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
