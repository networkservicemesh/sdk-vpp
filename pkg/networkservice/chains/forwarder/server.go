// Copyright (c) 2020-2023 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2023 Nordix Foundation.
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

package forwarder

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipsecapi "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/ipsec"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/cleanup"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismpriority"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	authmonitor "github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/ethernetcontext"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/afxdppinhole"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/mtu"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/ipsec"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/wireguard"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/nsmonitor"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/pinhole"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/stats"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/tag"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect/l2bridgedomain"
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
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, vppConn Connection, tunnelIP net.IP, options ...Option) endpoint.Endpoint {
	opts := &forwarderOptions{
		name:                             "forwarder-vpp-" + uuid.New().String(),
		authorizeServer:                  authorize.NewServer(authorize.Any()),
		authorizeMonitorConnectionServer: authmonitor.NewMonitorConnectionServer(authmonitor.Any()),
		clientURL:                        &url.URL{Scheme: "unix", Host: "connect.to.socket"},
		dialTimeout:                      time.Millisecond * 200,
		domain2Device:                    make(map[string]string),
	}
	for _, opt := range options {
		opt(opts)
	}
	nseClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithClientURL(opts.clientURL),
		registryclient.WithNSEAdditionalFunctionality(
			registryrecvfd.NewNetworkServiceEndpointRegistryClient(),
			registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		),
		registryclient.WithDialOptions(opts.dialOpts...),
	)
	nsClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithClientURL(opts.clientURL),
		registryclient.WithDialOptions(opts.dialOpts...))

	rv := &xconnectNSServer{}
	pinholeMutex := new(sync.Mutex)
	additionalFunctionality := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		sendfd.NewServer(),
		discover.NewServer(nsClient, nseClient),
		roundrobin.NewServer(),
		stats.NewServer(ctx, opts.statsOpts...),
		up.NewServer(ctx, vppConn),
		xconnect.NewServer(vppConn),
		l2bridgedomain.NewServer(vppConn),
		connectioncontextkernel.NewServer(),
		ethernetcontext.NewVFServer(),
		tag.NewServer(ctx, vppConn),
		mtu.NewServer(vppConn),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			memif.MECHANISM: memif.NewServer(ctx, vppConn,
				memif.WithDirectMemif(),
				memif.WithChangeNetNS()),
			kernel.MECHANISM:    kernel.NewServer(vppConn),
			vxlan.MECHANISM:     vxlan.NewServer(vppConn, tunnelIP, opts.vxlanOpts...),
			wireguard.MECHANISM: wireguard.NewServer(vppConn, tunnelIP),
			ipsecapi.MECHANISM:  ipsec.NewServer(vppConn, tunnelIP),
		}),
		afxdppinhole.NewServer(),
		pinhole.NewServer(vppConn, pinhole.WithSharedMutex(pinholeMutex)),
		connect.NewServer(
			client.NewClient(ctx,
				client.WithoutRefresh(),
				client.WithName(opts.name),
				client.WithDialOptions(opts.dialOpts...),
				client.WithDialTimeout(opts.dialTimeout),
				client.WithAdditionalFunctionality(
					append([]networkservice.NetworkServiceClient{
						cleanup.NewClient(ctx, opts.cleanupOpts...),
						mechanismtranslation.NewClient(),
						connectioncontextkernel.NewClient(),
						stats.NewClient(ctx, opts.statsOpts...),
						up.NewClient(ctx, vppConn),
						mtu.NewClient(vppConn),
						tag.NewClient(ctx, vppConn),
						// mechanisms
						memif.NewClient(ctx, vppConn,
							memif.WithChangeNetNS(),
						),
						kernel.NewClient(vppConn),
						vxlan.NewClient(vppConn, tunnelIP, opts.vxlanOpts...),
						wireguard.NewClient(vppConn, tunnelIP),
						ipsec.NewClient(vppConn, tunnelIP),
						vlan.NewClient(vppConn, opts.domain2Device),
						filtermechanisms.NewClient(),
						mechanismpriority.NewClient(opts.mechanismPrioriyList...),
						afxdppinhole.NewClient(),
						pinhole.NewClient(vppConn, pinhole.WithSharedMutex(pinholeMutex)),
						recvfd.NewClient(),
						nsmonitor.NewClient(ctx),
						sendfd.NewClient(),
					},
						opts.clientAdditionalFunctionality...,
					)...,
				),
			),
		),
	}

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAuthorizeMonitorConnectionServer(opts.authorizeMonitorConnectionServer),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	return rv
}
