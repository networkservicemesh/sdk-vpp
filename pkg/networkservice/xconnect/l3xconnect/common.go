// Copyright (c) 2021 Cisco and/or its affiliates.
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

package l3xconnect

import (
	"context"
	"net"
	"time"

	"github.com/networkservicemesh/govpp/binapi/fib_types"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/l3xc"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

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

	clientNextHops := conn.GetContext().GetIpContext().GetSrcIPNets()
	serverNextHops := conn.GetContext().GetIpContext().GetDstIPNets()

	for _, update := range l3xcUpdates(clientIfIndex, serverIfIndex, clientNextHops, serverNextHops) {
		now := time.Now()
		if _, err := l3xc.NewServiceClient(vppConn).L3xcUpdate(ctx, update); err != nil {
			return errors.Wrap(err, "vppapi L3xcUpdate returned error")
		}
		log.FromContext(ctx).
			WithField("SwIfIndex", update.L3xc.SwIfIndex).
			WithField("IsIP6", update.L3xc.IsIP6).
			WithField("Paths[0].SwIfIndex", update.L3xc.Paths[0].SwIfIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "L3xcUpdate").Debug("completed")
	}

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
	for _, ifIndex := range []interface_types.InterfaceIndex{clientIfIndex, serverIfIndex} {
		for _, isIP6 := range []bool{true, false} {
			now := time.Now()
			if _, err := l3xc.NewServiceClient(vppConn).L3xcDel(ctx, &l3xc.L3xcDel{
				SwIfIndex: ifIndex,
				IsIP6:     isIP6,
			}); err != nil {
				return errors.Wrap(err, "vppapi L3xcDel returned error")
			}
			log.FromContext(ctx).
				WithField("SwIfIndex", ifIndex).
				WithField("IsIP6", isIP6).
				WithField("duration", time.Since(now)).
				WithField("vppapi", "L3xcDel").Debug("completed")
		}
	}
	return nil
}

func l3xcUpdates(clientSwIfIndex, serverSwIfIndex interface_types.InterfaceIndex, clientNextHops, serverNextHops []*net.IPNet) []*l3xc.L3xcUpdate {
	return []*l3xc.L3xcUpdate{
		l3xcUpdate(clientSwIfIndex, serverSwIfIndex, clientNextHops, false),
		l3xcUpdate(clientSwIfIndex, serverSwIfIndex, clientNextHops, true),
		l3xcUpdate(serverSwIfIndex, clientSwIfIndex, serverNextHops, false),
		l3xcUpdate(serverSwIfIndex, clientSwIfIndex, serverNextHops, true),
	}
}

func l3xcUpdate(fromSwIfIndex, toIfIndex interface_types.InterfaceIndex, nextHops []*net.IPNet, isIP6 bool) *l3xc.L3xcUpdate {
	rv := &l3xc.L3xcUpdate{
		L3xc: l3xc.L3xc{
			SwIfIndex: fromSwIfIndex,
			IsIP6:     isIP6,
		},
	}
	for _, nh := range nextHops {
		if nh == nil {
			continue
		}
		if isIP6 && nh.IP.To4() != nil {
			continue
		}
		proto := fib_types.FIB_API_PATH_NH_PROTO_IP4
		if isIP6 {
			proto = fib_types.FIB_API_PATH_NH_PROTO_IP6
		}
		rv.L3xc.NPaths++
		rv.L3xc.Paths = append(rv.L3xc.Paths, fib_types.FibPath{
			SwIfIndex: uint32(toIfIndex),
			Proto:     proto,
			Nh: fib_types.FibPathNh{
				Address: types.ToVppAddress(nh.IP).Un,
			},
		})
		break
	}
	if rv.L3xc.NPaths == 0 {
		rv.L3xc.NPaths = 1
		proto := fib_types.FIB_API_PATH_NH_PROTO_IP4
		if isIP6 {
			proto = fib_types.FIB_API_PATH_NH_PROTO_IP6
		}
		rv.L3xc.Paths = []fib_types.FibPath{
			{
				SwIfIndex: uint32(toIfIndex),
				Proto:     proto,
			},
		}
	}
	return rv
}
