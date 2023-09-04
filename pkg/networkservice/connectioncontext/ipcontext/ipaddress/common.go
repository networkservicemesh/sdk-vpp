// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
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

package ipaddress

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, loadIfIndex ifIndexFunc, isClient, isAdd bool) error {
	swIfIndex, ok := loadIfIndex(ctx, isClient)
	if !ok {
		return errors.New("no swIfIndex available")
	}
	ipNets := conn.GetContext().GetIpContext().GetDstIPNets()
	if isClient {
		ipNets = conn.GetContext().GetIpContext().GetSrcIPNets()
	}
	if ipNets == nil {
		return nil
	}

	var curIPs []net.IP
	if isAdd {
		var err error
		if curIPs, err = dumpIps(ctx, vppConn, swIfIndex); err != nil {
			return err
		}
	}
	for _, ipNet := range ipNets {
		// Сheck if the interface already has ipNet
		if isAdd {
			has := false
			for _, addr := range curIPs {
				if addr.Equal(ipNet.IP) {
					has = true
					break
				}
			}
			if has {
				continue
			}
		}
		now := time.Now()
		if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceAddDelAddress(ctx, &interfaces.SwInterfaceAddDelAddress{
			SwIfIndex: swIfIndex,
			IsAdd:     isAdd,
			Prefix:    types.ToVppAddressWithPrefix(ipNet),
		}); err != nil {
			return errors.Wrap(err, "vppapi SwInterfaceAddDelAddress returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", swIfIndex).
			WithField("prefix", ipNets).
			WithField("isAdd", isAdd).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceAddDelAddress").Debug("completed")
	}
	return nil
}

func dumpIps(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) ([]net.IP, error) {
	var ips []net.IP
	for _, isIPv6 := range []bool{false, true} {
		ipAddressClient, err := ip.NewServiceClient(vppConn).IPAddressDump(ctx, &ip.IPAddressDump{
			SwIfIndex: swIfIndex,
			IsIPv6:    isIPv6,
		})
		if err != nil {
			return nil, errors.Wrap(err, "vppapi IPAddressDump returned error")
		}
		for {
			ipAddressDetails, err := ipAddressClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			ips = append(ips, types.FromVppAddressWithPrefix(ipAddressDetails.Prefix).IP)
		}
	}
	return ips, nil
}
