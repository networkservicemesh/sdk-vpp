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

package mtu

import (
	"context"
	"io"
	"net"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/ip"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func getMTU(ctx context.Context, vppConn api.Connection, tunnelIP net.IP) (uint32, error) {
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if err != nil {
		return 0, errors.Wrapf(err, "error attempting to get interface dump client to determine MTU for tunnelIP %q", tunnelIP)
	}
	defer func() { _ = client.Close() }()

	for {
		details, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrapf(err, "error attempting to get interface details to determine MTU for tunnelIP %q", tunnelIP)
		}

		ipAddressClient, err := ip.NewServiceClient(vppConn).IPAddressDump(ctx, &ip.IPAddressDump{
			SwIfIndex: details.SwIfIndex,
			IsIPv6:    tunnelIP.To4() == nil,
		})
		if err != nil {
			return 0, errors.Wrapf(err, "error attempting to get ip address for vpp interface %q determine MTU for tunnelIP %q", details.InterfaceName, tunnelIP)
		}
		defer func() { _ = ipAddressClient.Close() }()

		for {
			ipAddressDetails, err := ipAddressClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, errors.Wrapf(err, "error attempting to get interface ip address for %q (swIfIndex: %q) to determine MTU for tunnelIP %q", details.InterfaceName, details.SwIfIndex, tunnelIP)
			}
			if types.FromVppAddressWithPrefix(ipAddressDetails.Prefix).IP.Equal(tunnelIP) && details.Mtu[0] != 0 {
				return (details.Mtu[0] - overhead(tunnelIP.To4() == nil)), nil
			}
		}
	}
	return 0, errors.Errorf("unable to find interface in vpp with tunnelIP: %q or interface IP MTU is zero", tunnelIP)
}

func overhead(isV6 bool) uint32 {
	if !isV6 {
		// IP Header 	20
		// UDP Header 	8
		// IPsec Sequence Number 	4
		// IPsec SPI 	4
		// Initialization Vector 	16
		// Padding 	0 – 15
		// Padding Length 	1
		// Next Header 	1
		// Authentication Data 	12
		return 81
	}
	// IPv6 Header 	40
	// UDP Header 	8
	// IPsec Sequence Number 	4
	// IPsec SPI 	4
	// Initialization Vector 	16
	// Padding 	0 – 15
	// Padding Length 	1
	// Next Header 	1
	// Authentication Data 	12
	return 101
}
