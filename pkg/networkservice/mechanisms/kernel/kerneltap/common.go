// Copyright (c) 2020 Cisco and/or its affiliates.
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

package kerneltap

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/tapv2"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if _, ok := ifindex.Load(ctx, isClient); ok {
			return nil
		}
		now := time.Now()
		rsp, err := tapv2.NewServiceClient(vppConn).TapCreateV2(ctx, &tapv2.TapCreateV2{
			ID:            ^uint32(0),
			UseRandomMac:  true,
			NumRxQueues:   1,
			HostIfNameSet: true,
			HostIfName:    linuxIfaceName(mechanism.GetInterfaceName(conn)),
			//TapFlags:         0, // TODO - TUN support for v3 payloads
		})
		if err != nil {
			return errors.WithStack(err)
		}
		trace.Log(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "TapCreateV2").Debug("completed")
		ifindex.Store(ctx, isClient, rsp.SwIfIndex)

		now = time.Now()
		if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetRxMode(ctx, &interfaces.SwInterfaceSetRxMode{
			SwIfIndex: rsp.SwIfIndex,
			Mode:      interface_types.RX_MODE_API_ADAPTIVE,
		}); err != nil {
			return errors.WithStack(err)
		}
		trace.Log(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceSetRxMode").Debug("completed")
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		swIfIndex, ok := ifindex.Load(ctx, isClient)
		if !ok {
			return nil
		}
		_, err := tapv2.NewServiceClient(vppConn).TapDeleteV2(ctx, &tapv2.TapDeleteV2{
			SwIfIndex: swIfIndex,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	return nil
}

func linuxIfaceName(ifaceName string) string {
	if len(ifaceName) <= kernel.LinuxIfMaxLength {
		return ifaceName
	}
	return ifaceName[:kernel.LinuxIfMaxLength]
}
