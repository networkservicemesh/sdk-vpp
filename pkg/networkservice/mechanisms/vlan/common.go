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

package vlan

import (
	"context"
	"io"
	"time"

	"git.fd.io/govpp.git/api"

	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func addSubIf(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, deviceNames map[string]string) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		_, ok := ifindex.Load(ctx, true)
		if ok {
			return nil
		}
		now := time.Now()
		via := conn.GetLabels()[viaLabel]
		hostIFName, ok := deviceNames[via]
		if !ok {
			return errors.Errorf("no interface name for service domain %s", via)
		}

		client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{
			NameFilterValid: true,
			NameFilter:      hostIFName,
		})
		if err != nil {
			return errors.Wrapf(err, "error attempting to get interface dump client to set vlan subinterface on %q", hostIFName)
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
				return errors.Wrapf(err, "error attempting to get interface details to set vlan subinterface on %q", hostIFName)
			}
			now = time.Now()
			swIfIndex := details.SwIfIndex
			vlanID := mechanism.GetVlanID()
			vlanSubif := &interfaces.CreateVlanSubif{
				SwIfIndex: swIfIndex,
				VlanID:    vlanID,
			}

			rsp, err := interfaces.NewServiceClient(vppConn).CreateVlanSubif(ctx, vlanSubif)
			if err != nil {
				return errors.WithStack(err)
			}
			log.FromContext(ctx).
				WithField("duration", time.Since(now)).
				WithField("HostInterfaceIndex", swIfIndex).
				WithField("VlanID", vlanID).
				WithField("vppapi", "CreateVlanSubIf").Debug("completed")

			ifindex.Store(ctx, true, rsp.SwIfIndex)
		}
	}
	return nil
}
func delSubIf(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		swIfIndex, ok := ifindex.Load(ctx, true)
		if !ok {
			return nil
		}
		now := time.Now()
		vlanSubif := &interfaces.DeleteSubif{
			SwIfIndex: swIfIndex,
		}
		_, err := interfaces.NewServiceClient(vppConn).DeleteSubif(ctx, vlanSubif)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("duration", time.Since(now)).
			WithField("HostInterfaceIndex", swIfIndex).
			WithField("vppapi", "DeleteSubif").Debug("completed")
		ifindex.Delete(ctx, true)
	}
	return nil
}
