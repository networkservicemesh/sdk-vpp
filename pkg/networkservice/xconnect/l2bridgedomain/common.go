// Copyright (c) 2022 Nordix Foundation.
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

package l2bridgedomain

import (
	"context"
	"time"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/l2"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type bridgeDomain struct {
	// BdID
	id uint32

	// attached interfaces
	attached map[interface_types.InterfaceIndex]struct{}
}

type bridgeDomainKey struct {
	vlanID        uint32
	clientIfIndex interface_types.InterfaceIndex
}

func addBridgeDomain(ctx context.Context, vppConn api.Connection, bridges *l2BridgeDomain, vlanID uint32) error {
	clientIfIndex, ok := ifindex.Load(ctx, true)
	if !ok {
		return nil
	}
	serverIfIndex, ok := ifindex.Load(ctx, false)
	if !ok {
		return nil
	}

	// key <vlanID, clientIfIndex> to handle the vlanID == 0 case
	key := bridgeDomainKey{
		vlanID:        vlanID,
		clientIfIndex: clientIfIndex,
	}
	l2Bridge, ok := bridges.Load(key)
	if !ok {
		bridgeID, err := addDelVppBridgeDomain(ctx, vppConn, ^uint32(0), true)
		if err != nil {
			return err
		}
		l2Bridge = &bridgeDomain{
			id:       bridgeID,
			attached: make(map[interface_types.InterfaceIndex]struct{}),
		}
		bridges.Store(key, l2Bridge)
	}
	if _, ok = l2Bridge.attached[serverIfIndex]; !ok {
		err := addDelVppInterfaceBridgeDomain(ctx, vppConn, serverIfIndex, l2Bridge.id, 1, true)
		if err != nil {
			return err
		}
		l2Bridge.attached[serverIfIndex] = struct{}{}
		bridges.Store(key, l2Bridge)
	}
	if _, ok = l2Bridge.attached[clientIfIndex]; !ok {
		err := addDelVppInterfaceBridgeDomain(ctx, vppConn, clientIfIndex, l2Bridge.id, 0, true)
		if err != nil {
			return err
		}
		l2Bridge.attached[clientIfIndex] = struct{}{}
		bridges.Store(key, l2Bridge)
	}
	return nil
}

func delBridgeDomain(ctx context.Context, vppConn api.Connection, bridges *l2BridgeDomain, vlanID uint32) error {
	if clientIfIndex, ok := ifindex.Load(ctx, true); ok {
		key := bridgeDomainKey{
			vlanID:        vlanID,
			clientIfIndex: clientIfIndex,
		}
		l2Bridge, ok := bridges.Load(key)
		if !ok {
			return nil
		}
		if serverIfIndex, okey := ifindex.Load(ctx, false); okey {
			if _, ok = l2Bridge.attached[serverIfIndex]; ok {
				err := addDelVppInterfaceBridgeDomain(ctx, vppConn, serverIfIndex, l2Bridge.id, 0, false)
				if err != nil {
					return err
				}
				delete(l2Bridge.attached, serverIfIndex)
			}
		}
		if len(l2Bridge.attached) == 1 {
			// last interface -> delete the bridge and the sub-interface also
			if _, ok = l2Bridge.attached[clientIfIndex]; ok {
				err := addDelVppInterfaceBridgeDomain(ctx, vppConn, clientIfIndex, l2Bridge.id, 0, false)
				if err != nil {
					return err
				}
				err = delVppSubIf(ctx, vppConn, vlanID, clientIfIndex)
				if err != nil {
					return err
				}
				delete(l2Bridge.attached, clientIfIndex)
				_, err = addDelVppBridgeDomain(ctx, vppConn, l2Bridge.id, false)
				if err != nil {
					return err
				}
				bridges.Delete(key)
			}
		} else {
			bridges.Store(key, l2Bridge)
		}
	}
	return nil
}

func addDelVppBridgeDomain(ctx context.Context, vppConn api.Connection, bridgeID uint32, isAdd bool) (uint32, error) {
	now := time.Now()
	bridgeDomainAddDelV2 := &l2.BridgeDomainAddDelV2{
		IsAdd:   isAdd,
		BdID:    bridgeID,
		Flood:   true,
		Forward: true,
		Learn:   true,
		UuFlood: true,
	}
	rsp, err := l2.NewServiceClient(vppConn).BridgeDomainAddDelV2(ctx, bridgeDomainAddDelV2)
	if err != nil {
		return 0, errors.Wrap(err, "vppapi BridgeDomainAddDelV2 returned error")
	}
	log.FromContext(ctx).
		WithField("bridgeID", rsp.BdID).
		WithField("isAdd", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "BridgeDomainAddDelV2").Info("completed")
	return rsp.BdID, nil
}

func addDelVppInterfaceBridgeDomain(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, bridgeID uint32, shg uint8, isAdd bool) error {
	now := time.Now()
	_, err := l2.NewServiceClient(vppConn).SwInterfaceSetL2Bridge(ctx, &l2.SwInterfaceSetL2Bridge{
		RxSwIfIndex: swIfIndex,
		Enable:      isAdd,
		BdID:        bridgeID,
		Shg:         shg,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi SwInterfaceSetL2Bridge returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("bridgeID", bridgeID).
		WithField("isAdd", isAdd).
		WithField("shg", shg).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceSetL2Bridge").Info("completed")
	return nil
}

func delVppSubIf(ctx context.Context, vppConn api.Connection, vlanID uint32, swIfIndex interface_types.InterfaceIndex) error {
	if vlanID == 0 {
		ifindex.Delete(ctx, true)
		return nil
	}
	now := time.Now()
	vlanSubif := &interfaces.DeleteSubif{
		SwIfIndex: swIfIndex,
	}
	_, err := interfaces.NewServiceClient(vppConn).DeleteSubif(ctx, vlanSubif)
	if err != nil {
		return errors.Wrap(err, "vppapi DeleteSubif returned error")
	}
	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("InterfaceIndex", swIfIndex).
		WithField("vppapi", "DeleteSubif").Debug("completed")
	return nil
}
