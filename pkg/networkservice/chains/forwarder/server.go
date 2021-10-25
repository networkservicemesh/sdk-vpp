// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

package forwarder

import (
	"context"
	"net"
	"net/url"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/ethernetcontext"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/mtu"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/wireguard"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/pinhole"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/stats"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/tag"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
)

// Connection aggregates the api.Connection and api.ChannelProvider interfaces
type Connection interface {
	IsExternal() bool

	api.Connection
	api.ChannelProvider
}

type xconnectNSServer struct {
	endpoint.Endpoint
}

// NewServer - returns an implementation of the xconnectns network service
func NewServer(ctx context.Context, name string, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, clientURL *url.URL, vppConn Connection, tunnelIP net.IP, tunnelPort uint16, dialTimeout time.Duration, domain2Device map[string]string, clientDialOptions ...grpc.DialOption) endpoint.Endpoint {
	nseClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx, clientURL,
		registryclient.WithNSEAdditionalFunctionality(registryrecvfd.NewNetworkServiceEndpointRegistryClient()),
		registryclient.WithDialOptions(clientDialOptions...),
	)
	nsClient := registryclient.NewNetworkServiceRegistryClient(ctx, clientURL, registryclient.WithDialOptions(clientDialOptions...))

	rv := &xconnectNSServer{}
	additionalFunctionality := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		sendfd.NewServer(),
		discover.NewServer(nsClient, nseClient),
		roundrobin.NewServer(),
		stats.NewServer(ctx),
		up.NewServer(ctx, vppConn),
		xconnect.NewServer(vppConn),
		connectioncontextkernel.NewServer(),
		ethernetcontext.NewVFServer(),
		tag.NewServer(ctx, vppConn),
		mtu.NewServer(vppConn),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			memif.MECHANISM: memif.NewServer(ctx, vppConn,
				memif.WithDirectMemif(),
				memif.WithChangeNetNS()),
			kernel.MECHANISM:    kernel.NewServer(vppConn),
			vxlan.MECHANISM:     vxlan.NewServer(vppConn, tunnelIP, vxlan.WithVniPort(tunnelPort)),
			wireguard.MECHANISM: wireguard.NewServer(vppConn, tunnelIP),
		}),
		pinhole.NewServer(vppConn),
		connect.NewServer(
			client.NewClient(ctx,
				client.WithoutRefresh(),
				client.WithName(name),
				client.WithDialOptions(clientDialOptions...),
				client.WithDialTimeout(dialTimeout),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
					connectioncontextkernel.NewClient(),
					stats.NewClient(ctx),
					up.NewClient(ctx, vppConn),
					mtu.NewClient(vppConn),
					tag.NewClient(ctx, vppConn),
					// mechanisms
					memif.NewClient(vppConn,
						memif.WithChangeNetNS()),
					kernel.NewClient(vppConn),
					vxlan.NewClient(vppConn, tunnelIP, vxlan.WithVniPort(tunnelPort)),
					wireguard.NewClient(vppConn, tunnelIP),
					vlan.NewClient(vppConn, domain2Device),
					filtermechanisms.NewClient(),
					pinhole.NewClient(vppConn),
					recvfd.NewClient(),
					sendfd.NewClient()),
			),
		),
	}

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authzServer),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	return rv
}
