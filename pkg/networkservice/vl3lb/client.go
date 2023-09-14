// Copyright (c) 2023 Cisco and/or its affiliates.
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

package vl3lb

import (
	"context"
	"net"
	"net/url"
	"time"

	"go.fd.io/govpp/api"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/govpp/binapi/ip_types"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
)

type vl3lbClient struct {
	port       uint16
	targetPort uint16
	protocol   ip_types.IPProto
	selector   map[string]string

	chainCtx          context.Context
	vppConn           api.Connection
	nseRegistryClient registry.NetworkServiceEndpointRegistryClient
	dialTimeout       time.Duration
	dialOpts          []grpc.DialOption
}

// NewClient - return a new Client chain element implementing the vl3 load balancer
func NewClient(chainCtx context.Context, vppConn api.Connection, options ...Option) networkservice.NetworkServiceClient {
	opts := &vl3LBOptions{
		port:       80,
		targetPort: 80,
		protocol:   ip_types.IP_API_PROTO_TCP,
		selector:   make(map[string]string),
		clientURL:  &url.URL{Scheme: "unix", Host: "connect.to.socket"},
	}
	for _, opt := range options {
		opt(opts)
	}

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(chainCtx,
		registryclient.WithClientURL(opts.clientURL),
		registryclient.WithNSEAdditionalFunctionality(
			registryrecvfd.NewNetworkServiceEndpointRegistryClient(),
			registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		),
		registryclient.WithDialOptions(opts.dialOpts...),
	)

	return &vl3lbClient{
		port:              opts.port,
		targetPort:        opts.targetPort,
		protocol:          opts.protocol,
		selector:          opts.selector,
		chainCtx:          chainCtx,
		vppConn:           vppConn,
		nseRegistryClient: nseRegistryClient,
		dialTimeout:       opts.dialTimeout,
		dialOpts:          opts.dialOpts,
	}
}

func (lb *vl3lbClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	requestContext := request.GetConnection().GetContext()
	var previousIP net.IP
	if requestContext != nil && requestContext.GetIpContext() != nil {
		previousIP = requestContext.GetIpContext().GetSrcIPNets()[0].IP
	}
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if !previousIP.Equal(conn.GetContext().GetIpContext().GetSrcIPNets()[0].IP) {
		if cancel, ok := loadAndDeleteCancel(ctx); ok {
			cancel()
		}
		cancelCtx, cancel := context.WithCancel(lb.chainCtx)
		go lb.balanceService(cancelCtx, conn)

		storeCancel(ctx, cancel)
	}

	return conn, nil
}

func (lb *vl3lbClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cancel, ok := loadAndDeleteCancel(ctx); ok {
		cancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
