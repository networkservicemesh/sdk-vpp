// Copyright (c) 2020-2024 Cisco and/or its affiliates.
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

package mtu

import (
	"context"
	"io"
	"net"

	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/ip"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func getMTU(ctx context.Context, vppConn api.Connection, tunnelIP net.IP) (uint32, error) {
	newCtx := mechutils.ToSafeContext(ctx)
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(newCtx, &interfaces.SwInterfaceDump{})
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

		ipAddressClient, err := ip.NewServiceClient(vppConn).IPAddressDump(newCtx, &ip.IPAddressDump{
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
	// https://lists.zx2c4.com/pipermail/wireguard/2017-December/002201.html
	if !isV6 {
		//  20-byte outer IPv4 header
		//  8-byte outer UDP header
		//  4-byte type
		//  4-byte key index
		//  8-byte nonce
		// 16-byte authentication tag
		// 60 byte total
		return 60
	}
	//  40-byte outer IPv4 header
	//  8-byte outer UDP header
	//  4-byte type
	//  4-byte key index
	//  8-byte nonce
	// 16-byte authentication tag
	// 80 bytes total
	return 80
}
