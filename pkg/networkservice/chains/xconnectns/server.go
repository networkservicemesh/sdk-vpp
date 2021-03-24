// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package xconnectns

import (
	"context"
	"net"
	"net/url"

	"git.fd.io/govpp.git/api"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontextkernel"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/stats"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/tag"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect/l2xconnect"
)

// Connection aggregates the api.Connection and api.ChannelProvider interfaces
type Connection interface {
	api.Connection
	api.ChannelProvider
}

type xconnectNSServer struct {
	endpoint.Endpoint
}

// NewServer - returns an implementation of the xconnectns network service
func NewServer(ctx context.Context, name string, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, clientURL *url.URL, vppConn Connection, tunnelIP net.IP, clientDialOptions ...grpc.DialOption) endpoint.Endpoint {
	rv := &xconnectNSServer{}
	additionalFunctionality := []networkservice.NetworkServiceServer{
		metadata.NewServer(),
		recvfd.NewServer(),
		stats.NewServer(ctx),
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(clientURL),
		heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
		connect.NewServer(
			ctx,
			func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return client.NewClient(ctx,
					cc,
					client.WithName(name),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						connectioncontextkernel.NewClient(),
						stats.NewClient(ctx),
						tag.NewClient(ctx, vppConn),
						// mechanisms
						memif.NewClient(vppConn),
						kernel.NewClient(vppConn),
						vxlan.NewClient(vppConn, tunnelIP),
						recvfd.NewClient(),
						sendfd.NewClient(),
					),
				)
			},
			connect.WithDialOptions(clientDialOptions...),
		),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			memif.MECHANISM:  memif.NewServer(vppConn),
			kernel.MECHANISM: kernel.NewServer(vppConn),
			vxlan.MECHANISM:  vxlan.NewServer(vppConn, tunnelIP),
		}),
		tag.NewServer(ctx, vppConn),
		connectioncontextkernel.NewServer(),
		l2xconnect.NewServer(vppConn),
		up.NewServer(ctx, vppConn),
		sendfd.NewServer(),
	}

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authzServer),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	return rv
}
