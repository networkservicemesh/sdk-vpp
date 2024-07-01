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

package kerneltap

import (
	"context"
	"time"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/tapv2"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	kernellink "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
)

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// Construct the netlink handle for the target namespace for this kernel interface
		handle, err := kernellink.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return err
		}
		defer handle.Close()

		if _, ok := ifindex.Load(ctx, isClient); ok {
			if _, err = handle.LinkByName(mechanism.GetInterfaceName()); err == nil {
				return nil
			}
		}
		// Delete the kernel interface if there is one in the target namespace
		_ = del(ctx, conn, vppConn, isClient)

		nsFilename, err := mechutils.ToNSFilename(mechanism)
		if err != nil {
			return err
		}

		now := time.Now()
		tapCreate := &tapv2.TapCreateV3{
			ID:               ^uint32(0),
			UseRandomMac:     true,
			NumRxQueues:      1,
			HostIfNameSet:    true,
			HostIfName:       mechanism.GetInterfaceName(),
			HostNamespaceSet: true,
			HostNamespace:    nsFilename,
			TapFlags:         tapv2.TAP_API_FLAG_TUN,
		}

		if conn.GetPayload() == payload.Ethernet {
			tapCreate.TapFlags ^= tapv2.TAP_API_FLAG_TUN
		}

		rsp, err := tapv2.NewServiceClient(vppConn).TapCreateV3(ctx, tapCreate)
		if err != nil {
			return errors.Wrap(err, "vppapi TapCreateV3 returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("HostIfName", tapCreate.HostIfName).
			WithField("HostNamespace", tapCreate.HostNamespace).
			WithField("TapFlags", tapCreate.TapFlags).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "TapCreateV3").Debug("completed")
		ifindex.Store(ctx, isClient, rsp.SwIfIndex)

		now = time.Now()
		if _, err = interfaces.NewServiceClient(vppConn).SwInterfaceSetRxMode(ctx, &interfaces.SwInterfaceSetRxMode{
			SwIfIndex: rsp.SwIfIndex,
			Mode:      interface_types.RX_MODE_API_ADAPTIVE,
		}); err != nil {
			return errors.Wrap(err, "vppapi SwInterfaceSetRxMode returned error")
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceSetRxMode").Debug("completed")

		now = time.Now()
		l, err := handle.LinkByName(tapCreate.HostIfName)
		if err != nil {
			return errors.Wrapf(err, "unable to find hostIfName %s", tapCreate.HostIfName)
		}
		log.FromContext(ctx).
			WithField("link.Name", tapCreate.HostIfName).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkByName").Debug("completed")

		alias := mechutils.ToAlias(conn, isClient)

		// Set the Link Alias
		now = time.Now()
		if err = handle.LinkSetAlias(l, alias); err != nil {
			return errors.Wrapf(err, "failed to set the alias(%s) of the link device(%v)", alias, l)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("alias", alias).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetAlias").Debug("completed")

		// Up the link
		now = time.Now()
		err = handle.LinkSetUp(l)
		if err != nil {
			return errors.Wrapf(err, "failed to enable the link device: %v", l)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetUp").Debug("completed")
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		swIfIndex, ok := ifindex.LoadAndDelete(ctx, isClient)
		if !ok {
			return nil
		}
		now := time.Now()
		_, err := tapv2.NewServiceClient(vppConn).TapDeleteV2(context.Background(), &tapv2.TapDeleteV2{
			SwIfIndex: swIfIndex,
		})
		if err != nil {
			return errors.Wrapf(err, "unable to delete connection with SwIfIndex %v", swIfIndex)
		}
		log.FromContext(ctx).
			WithField("SwIfIndex", swIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "TapDeleteV2").Debug("completed")
		return nil
	}
	return nil
}
