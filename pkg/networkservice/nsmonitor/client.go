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

//go:build linux
// +build linux

package nsmonitor

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// Monitor provides interface for netns monitor
type Monitor interface {
	Watch(ctx context.Context, inodeURL string) <-chan struct{}
}

type netNSMonitorClient struct {
	chainCtx      context.Context
	monitor       Monitor
	onceInit      sync.Once
	supplyMonitor func(ctx context.Context) Monitor
}

// NewClient returns new net ns monitoring client
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	options := &clientOptions{
		supplyMonitor: newMonitor,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &netNSMonitorClient{
		chainCtx:      chainCtx,
		supplyMonitor: options.supplyMonitor,
	}
}

func (r *netNSMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	r.onceInit.Do(func() {
		r.monitor = r.supplyMonitor(r.chainCtx)
	})

	cancelCtx, cancel := context.WithCancel(r.chainCtx)
	if _, ok := metadata.Map(ctx, metadata.IsClient(r)).LoadOrStore(key{}, cancel); !ok {
		if inodeURL, ok := conn.GetMechanism().GetParameters()[common.InodeURL]; ok {
			deleteCh := r.monitor.Watch(cancelCtx, inodeURL)
			factory := begin.FromContext(ctx)
			go func() {
				select {
				case <-r.chainCtx.Done():
					return
				case <-cancelCtx.Done():
					return
				case _, ok := <-deleteCh:
					if ok {
						factory.Close(begin.CancelContext(cancelCtx))
					}
					return
				}
			}()
		}
	}

	return conn, nil
}

func (r *netNSMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if v, ok := metadata.Map(ctx, metadata.IsClient(r)).LoadAndDelete(key{}); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}
