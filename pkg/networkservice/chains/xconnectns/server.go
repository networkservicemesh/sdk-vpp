// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
//
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

package xconnectns

import (
	"context"
	"net"
	"net/url"
	"sync"

	"git.fd.io/govpp.git/api"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	noopmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/ethernetcontext"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/inject"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/noop"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resetmechanism"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/resourcepool"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov"
	"github.com/networkservicemesh/sdk-sriov/pkg/sriov/config"
	sriovtokens "github.com/networkservicemesh/sdk-sriov/pkg/tools/tokens"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/mtu"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
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
	api.Connection
	api.ChannelProvider
}

// NewServer - returns an implementation of the xconnectns network service supporting following mechanisms:
// * [local VFIO, local kernel with SR-IOV token] -> [remote NOOP]
// * [local memif, local kernel without SR-IOV token] -> [remote VXLAN, remote wireguard]
// * [remote VXLAN, remote wireguard] -> [local memif, local kernel]
// * [remote NOOP] -> [local NOOP]
func NewServer(
	ctx context.Context,
	name string,
	authzServer networkservice.NetworkServiceServer,
	tokenGenerator token.GeneratorFunc,
	vppConn Connection,
	tunnelIP net.IP,
	pciPool resourcepool.PCIPool,
	resourcePool resourcepool.ResourcePool,
	sriovConfig *config.Config,
	vfioDir, cgroupBaseDir string,
	clientURL *url.URL,
	clientDialOptions ...grpc.DialOption,
) endpoint.Endpoint {
	vppChain := createVPPChain(ctx, name, vppConn, tunnelIP, clientDialOptions)
	sriovChain := createSRIOVChain(ctx, name, pciPool, resourcePool, sriovConfig, vfioDir, cgroupBaseDir, clientDialOptions)
	noopChain := createNOOPChain(ctx, name, clientDialOptions)

	rv := new(struct{ endpoint.Endpoint })
	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authzServer),
		endpoint.WithAdditionalFunctionality(
			recvfd.NewServer(),
			sendfd.NewServer(),
			clienturl.NewServer(clientURL),
			heal.NewServer(ctx,
				heal.WithOnHeal(addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
				heal.WithOnRestore(heal.OnRestoreIgnore)),
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				vfiomech.MECHANISM: sriovChain,
				kernel.MECHANISM: switchcase.NewServer(
					&switchcase.ServerCase{
						Condition: func(_ context.Context, conn *networkservice.Connection) bool {
							return sriovtokens.IsTokenID(kernelmech.ToMechanism(conn.GetMechanism()).GetTokenID())
						},
						Server: sriovChain,
					},
					&switchcase.ServerCase{
						Condition: switchcase.Default,
						Server:    vppChain,
					},
				),
				memif.MECHANISM:     vppChain,
				vxlan.MECHANISM:     vppChain,
				wireguard.MECHANISM: vppChain,
				noopmech.MECHANISM:  noopChain,
			}),
		),
	)

	return rv
}

// createVPPChain implements:
// * [local memif, local kernel without SR-IOV token] -> [remote VXLAN, remote wireguard]
// * [remote VXLAN, remote wireguard] -> [local memif, local kernel]
func createVPPChain(
	ctx context.Context,
	name string,
	vppConn Connection,
	tunnelIP net.IP,
	clientDialOptions []grpc.DialOption,
) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		stats.NewServer(ctx),
		up.NewServer(ctx, vppConn),
		xconnect.NewServer(vppConn),
		connectioncontextkernel.NewServer(),
		tag.NewServer(ctx, vppConn),
		mtu.NewServer(vppConn),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			memif.MECHANISM:     memif.NewServer(vppConn, memif.WithDirectMemif()),
			kernel.MECHANISM:    kernel.NewServer(vppConn),
			vxlan.MECHANISM:     vxlan.NewServer(vppConn, tunnelIP),
			wireguard.MECHANISM: wireguard.NewServer(vppConn, tunnelIP),
		}),
		pinhole.NewServer(vppConn),
		connect.NewServer(ctx,
			client.NewClientFactory(
				client.WithName(name),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
					connectioncontextkernel.NewClient(),
					stats.NewClient(ctx),
					up.NewClient(ctx, vppConn),
					mtu.NewClient(vppConn),
					tag.NewClient(ctx, vppConn),
					// mechanisms
					memif.NewClient(vppConn),
					kernel.NewClient(vppConn),
					vxlan.NewClient(vppConn, tunnelIP),
					wireguard.NewClient(vppConn, tunnelIP),
					pinhole.NewClient(vppConn),
					recvfd.NewClient(),
					sendfd.NewClient(),
				),
			),
			connect.WithDialOptions(clientDialOptions...),
		),
	)
}

// createSRIOVChain implements:
// * [local VFIO, local kernel with SR-IOV token] -> [remote NOOP]
func createSRIOVChain(
	ctx context.Context,
	name string,
	pciPool resourcepool.PCIPool,
	resourcePool resourcepool.ResourcePool,
	sriovConfig *config.Config,
	vfioDir, cgroupBaseDir string,
	clientDialOptions []grpc.DialOption,
) networkservice.NetworkServiceServer {
	resourceLock := new(sync.Mutex)
	return chain.NewNetworkServiceServer(
		resetmechanism.NewServer(
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				kernel.MECHANISM: chain.NewNetworkServiceServer(
					resourcepool.NewServer(sriov.KernelDriver, resourceLock, pciPool, resourcePool, sriovConfig),
				),
				vfiomech.MECHANISM: chain.NewNetworkServiceServer(
					resourcepool.NewServer(sriov.VFIOPCIDriver, resourceLock, pciPool, resourcePool, sriovConfig),
					vfio.NewServer(vfioDir, cgroupBaseDir),
				),
			}),
		),
		// we setup VF ethernet context using PF interface, so we do it in the forwarder net NS
		ethernetcontext.NewVFServer(),
		inject.NewServer(),
		connectioncontextkernel.NewServer(),
		connect.NewServer(ctx,
			client.NewClientFactory(
				client.WithName(name),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
					noop.NewClient(cls.REMOTE),
				),
			),
			connect.WithDialOptions(clientDialOptions...),
		),
	)
}

// createNOOPChain implements:
// * [remote NOOP] -> [local NOOP]
func createNOOPChain(
	ctx context.Context,
	name string,
	clientDialOptions []grpc.DialOption,
) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		connect.NewServer(ctx,
			client.NewClientFactory(
				client.WithName(name),
				client.WithAdditionalFunctionality(
					mechanismtranslation.NewClient(),
					noop.NewClient(cls.LOCAL),
				),
			),
			connect.WithDialOptions(clientDialOptions...),
		),
	)
}
