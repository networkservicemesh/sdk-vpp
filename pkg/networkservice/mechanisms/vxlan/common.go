// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package vxlan

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/binapi/vpe"
	"github.com/edwarnicke/govpp/binapi/vxlan"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vxlanMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isAdd, isClient bool) error {
	if mechanism := vxlanMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		_, ok := ifindex.Load(ctx, isClient)
		if isAdd && ok {
			return nil
		}
		if !isAdd && !ok {
			return nil
		}
		if mechanism.SrcIP() == nil {
			return errors.Errorf("no vxlan SrcIP not provided")
		}
		if mechanism.DstIP() == nil {
			return errors.Errorf("no vxlan DstIP not provided")
		}

		now := time.Now()

		addNextNode := &vpe.AddNodeNext{
			NodeName: "vxlan4-input",
			NextName: "ethernet-input",
		}

		addNextNodeRsp, err := vpe.NewServiceClient(vppConn).AddNodeNext(ctx, addNextNode)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("NextIndex", addNextNodeRsp.NextIndex).
			WithField("NodeName", addNextNode.NodeName).
			WithField("NextName", addNextNode.NextName).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "AddNodeNext").Debug("completed")

		now = time.Now()
		vxlanAddDelTunnel := &vxlan.VxlanAddDelTunnel{
			IsAdd:          isAdd,
			Instance:       ^uint32(0),
			SrcAddress:     types.ToVppAddress(mechanism.SrcIP()),
			DstAddress:     types.ToVppAddress(mechanism.DstIP()),
			DecapNextIndex: addNextNodeRsp.NextIndex,
			Vni:            mechanism.VNI(),
		}
		if !isClient {
			vxlanAddDelTunnel.SrcAddress = types.ToVppAddress(mechanism.DstIP())
			vxlanAddDelTunnel.DstAddress = types.ToVppAddress(mechanism.SrcIP())
		}
		rsp, err := vxlan.NewServiceClient(vppConn).VxlanAddDelTunnel(ctx, vxlanAddDelTunnel)
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("swIfIndex", rsp.SwIfIndex).
			WithField("SrcAddress", vxlanAddDelTunnel.SrcAddress).
			WithField("DstAddress", vxlanAddDelTunnel.DstAddress).
			WithField("Vni", vxlanAddDelTunnel.Vni).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "VxlanAddDelTunnel").Debug("completed")
		ifindex.Store(ctx, isClient, rsp.SwIfIndex)
	}
	return nil
}
