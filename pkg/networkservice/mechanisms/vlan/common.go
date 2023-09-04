// Copyright (c) 2021-2022 Nordix Foundation.
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

package vlan

import (
	"context"
	"io"
	"strings"
	"time"

	"go.fd.io/govpp/api"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

const (
	afPacketNamePrefix = "host-"
)

func addSubIf(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, deviceNames map[string]string) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		_, ok := ifindex.Load(ctx, true)
		if ok {
			return nil
		}
		via := conn.GetLabels()[viaLabel]
		hostIFName, ok := deviceNames[via]
		if !ok {
			return errors.Errorf("no interface name for label %s", via)
		}
		vlanID := mechanism.GetVlanID()
		hostSwIfIndex, vlanSwIfIndex, err := getHostOrVlanInterface(ctx, vppConn, hostIFName, vlanID)
		if err != nil {
			return err
		}
		if vlanID != 0 {
			if vlanSwIfIndex != 0 {
				log.FromContext(ctx).
					WithField("VlanInterfaceIndex", vlanSwIfIndex).Debug("Vlan Interface already created")
				ifindex.Store(ctx, true, vlanSwIfIndex)
			} else {
				newVlanIfIndex, shouldReturn, returnValue := vppAddSubIf(ctx, vppConn, hostSwIfIndex, vlanID)
				if shouldReturn {
					return returnValue
				}
				ifindex.Store(ctx, true, *newVlanIfIndex)
			}
		} else {
			log.FromContext(ctx).
				WithField("HostInterfaceIndex", hostSwIfIndex).Debug("QinQ disabled")
			ifindex.Store(ctx, true, hostSwIfIndex)
		}
		/* Store vlanID used by bridge domain server */
		Store(ctx, true, vlanID)
	}
	return nil
}

func getHostOrVlanInterface(ctx context.Context, vppConn api.Connection, hostIFName string, vlanID uint32) (hostSwIfIndex, vlanSwIfIndex interface_types.InterfaceIndex, err error) {
	now := time.Now()
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{
		NameFilterValid: true,
		NameFilter:      hostIFName,
	})
	if err != nil {
		return 0, 0, errors.Wrapf(err, "error attempting to get interface dump client to set vlan subinterface on %q", hostIFName)
	}
	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("HostInterfaceName", hostIFName).
		WithField("vppapi", "SwInterfaceDump").Debug("completed")
	for {
		details, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, errors.Wrapf(err, "error attempting to get interface details to set vlan subinterface on %q", hostIFName)
		}
		if (vlanID != 0) && strings.Contains(details.InterfaceName, hostIFName) && (details.Type == interface_types.IF_API_TYPE_SUB) && (details.SubID == vlanID) {
			return 0, details.SwIfIndex, nil
		}
		if (hostIFName == details.InterfaceName) || (afPacketNamePrefix+hostIFName == details.InterfaceName) {
			hostSwIfIndex = details.SwIfIndex
		}
	}

	if hostSwIfIndex == 0 {
		return 0, 0, errors.Errorf("no interface name found %s", hostIFName)
	}
	return hostSwIfIndex, 0, nil
}

func vppAddSubIf(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, vlanID uint32) (*interface_types.InterfaceIndex, bool, error) {
	now := time.Now()
	vlanSubif := &interfaces.CreateVlanSubif{
		SwIfIndex: swIfIndex,
		VlanID:    vlanID,
	}

	rsp, err := interfaces.NewServiceClient(vppConn).CreateVlanSubif(ctx, vlanSubif)
	if err != nil {
		return nil, true, errors.Wrap(err, "vppapi CreateVlanSubif returned error")
	}
	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("HostInterfaceIndex", swIfIndex).
		WithField("SubInterfaceIndex", rsp.SwIfIndex).
		WithField("VlanID", vlanID).
		WithField("vppapi", "CreateVlanSubIf").Debug("completed")
	return &rsp.SwIfIndex, false, nil
}

func delSubIf(ctx context.Context, conn *networkservice.Connection) {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		_, ok := ifindex.Load(ctx, true)
		if !ok {
			return
		}
		/* Delete sub-interface together with the l2 bridge */
		ifindex.Delete(ctx, true)
		Delete(ctx, true)
	}
}
