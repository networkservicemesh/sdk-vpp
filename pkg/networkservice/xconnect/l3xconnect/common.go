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

package l3xconnect

import (
	"context"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/fib_types"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/l3xc"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func createXc(swIfIndexFrom, swIfIndexTo uint32, nextHop net.IP) l3xc.L3xc {
	isIP6 := nextHop.To4() == nil
	return l3xc.L3xc{
		SwIfIndex: interface_types.InterfaceIndex(swIfIndexFrom),
		IsIP6:     isIP6,
		NPaths:    1,
		Paths: []fib_types.FibPath{
			{
				SwIfIndex: swIfIndexTo,
				Weight:    1,
				Type:      fib_types.FIB_API_PATH_TYPE_NORMAL,
				Proto:     types.IsV6toFibProto(isIP6),
				Nh: fib_types.FibPathNh{
					Address: types.ToVppAddress(nextHop).Un,
				},
			},
		},
	}
}

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection) error {
	// L3xc works only with IP payload
	if conn.GetPayload() != payload.IP {
		return nil
	}

	clientIfIndex, ok := ifindex.Load(ctx, true)
	if !ok {
		return nil
	}
	serverIfIndex, ok := ifindex.Load(ctx, false)
	if !ok {
		return nil
	}

	nextHopSrcIP := conn.GetContext().GetIpContext().GetSrcIPNet().IP
	nextHopDstIP := conn.GetContext().GetIpContext().GetDstIPNet().IP
	isDirectOrder := conn.GetMechanism().GetCls() == cls.LOCAL

	// Set undefined index to allow vpp to choose an appropriate interface to remote by itself
	l3xcClient := createXc(uint32(clientIfIndex), uint32(serverIfIndex), nextHopSrcIP)
	l3xcServer := createXc(uint32(serverIfIndex), ^uint32(0), nextHopDstIP)
	if !isDirectOrder {
		l3xcClient = createXc(uint32(clientIfIndex), ^uint32(0), nextHopSrcIP)
		l3xcServer = createXc(uint32(serverIfIndex), uint32(clientIfIndex), nextHopDstIP)
	}

	// Create L3xc
	now := time.Now()
	_, err := l3xc.NewServiceClient(vppConn).L3xcUpdate(ctx, &l3xc.L3xcUpdate{
		L3xc: l3xcClient,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("RxSwIfIndex", clientIfIndex).
		WithField("TxSwIfIndex", serverIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcUpdate").Debug("completed")

	now = time.Now()
	_, err = l3xc.NewServiceClient(vppConn).L3xcUpdate(ctx, &l3xc.L3xcUpdate{
		L3xc: l3xcServer,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("RxSwIfIndex", serverIfIndex).
		WithField("TxSwIfIndex", clientIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcUpdate").Debug("completed")
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection) error {
	// L3xc works only with IP payload
	if conn.GetPayload() != payload.IP {
		return nil
	}
	isIP6 := conn.GetContext().GetIpContext().GetSrcIPNet().IP.To4() == nil

	now := time.Now()
	clientIfIndex, ok := ifindex.Load(ctx, true)
	if !ok {
		return nil
	}
	_, err := l3xc.NewServiceClient(vppConn).L3xcDel(ctx, &l3xc.L3xcDel{
		SwIfIndex: clientIfIndex,
		IsIP6:     isIP6,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", clientIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcDel").Debug("completed")

	now = time.Now()
	serverIfIndex, ok := ifindex.Load(ctx, false)
	if !ok {
		return nil
	}
	_, err = l3xc.NewServiceClient(vppConn).L3xcDel(ctx, &l3xc.L3xcDel{
		SwIfIndex: serverIfIndex,
		IsIP6:     isIP6,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", serverIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcDel").Debug("completed")

	return nil
}
