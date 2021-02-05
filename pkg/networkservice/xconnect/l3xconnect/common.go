// Copyright (c) 2021 Cisco and/or its affiliates.
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
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/fib_types"
	"github.com/edwarnicke/govpp/binapi/l3xc"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func create(ctx context.Context, vppConn api.Connection, conn *networkservice.Connection) error {
	clientIfIndex, ok := ifindex.Load(ctx, true)
	if !ok {
		return nil
	}
	serverIfIndex, ok := ifindex.Load(ctx, false)
	if !ok {
		return nil
	}

	now := time.Now()
	l3xcUpdate := &l3xc.L3xcUpdate{
		L3xc: l3xc.L3xc{
			SwIfIndex: clientIfIndex,
			NPaths:    1,
			Paths: []fib_types.FibPath{
				{
					SwIfIndex: uint32(serverIfIndex),
				},
			},
		},
	}
	if srcIPNet := conn.GetContext().GetIpContext().GetSrcIPNet(); srcIPNet != nil {
		l3xcUpdate.L3xc.Paths[0].Nh.Address = types.ToVppAddress(srcIPNet.IP).Un
	}
	if _, err := l3xc.NewServiceClient(vppConn).L3xcUpdate(ctx, l3xcUpdate); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", l3xcUpdate.L3xc.SwIfIndex).
		WithField("Paths[0].SwIfIndex", l3xcUpdate.L3xc.Paths[0].SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcUpdate").Debug("completed")

	now = time.Now()
	// TODO - handle delete case
	l3xcUpdate = &l3xc.L3xcUpdate{
		L3xc: l3xc.L3xc{
			SwIfIndex: serverIfIndex,
			NPaths:    1,
			Paths: []fib_types.FibPath{
				{
					SwIfIndex: uint32(clientIfIndex),
				},
			},
		},
	}
	if dstIPNet := conn.GetContext().GetIpContext().GetDstIPNet(); dstIPNet != nil {
		l3xcUpdate.L3xc.Paths[0].Nh.Address = types.ToVppAddress(dstIPNet.IP).Un
	}
	if _, err := l3xc.NewServiceClient(vppConn).L3xcUpdate(ctx, l3xcUpdate); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", serverIfIndex).
		WithField("Paths[0].SwIfIndex", clientIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcUpdate").Debug("completed")
	return nil
}

func del(ctx context.Context, vppConn api.Connection) error {
	clientIfIndex, ok := ifindex.Load(ctx, true)
	if !ok {
		return nil
	}
	serverIfIndex, ok := ifindex.Load(ctx, false)
	if !ok {
		return nil
	}

	now := time.Now()
	if _, err := l3xc.NewServiceClient(vppConn).L3xcDel(ctx, &l3xc.L3xcDel{
		SwIfIndex: clientIfIndex,
	}); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", clientIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcDel").Debug("completed")

	now = time.Now()
	if _, err := l3xc.NewServiceClient(vppConn).L3xcDel(ctx, &l3xc.L3xcDel{
		SwIfIndex: serverIfIndex,
	}); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", serverIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "L3xcDel").Debug("completed")
	return nil
}
