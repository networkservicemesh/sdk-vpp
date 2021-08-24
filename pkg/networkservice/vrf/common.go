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

package vrf

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/ip"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vrfInfo struct {
	/* vrf ID */
	id uint32

	/* count - the number of clients using this vrf ID */
	count uint32
}

type vrfMap struct {
	/* entries - is a map[NetworkServiceName]{vrfId, count} */
	entries map[string]*vrfInfo

	/* mutex for entries */
	mut sync.Mutex
}

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, t *vrfMap, isIPv6, isClient bool) (uint32, error) {
	t.mut.Lock()
	defer t.mut.Unlock()

	info, contains := t.entries[conn.NetworkService]
	if !contains {
		vrfID, err := createVPP(ctx, vppConn, isIPv6)
		if err != nil {
			return vrfID, err
		}
		info = &vrfInfo{
			id: vrfID,
		}
		t.entries[conn.NetworkService] = info
	}
	info.count++

	return info.id, nil
}

func createVPP(ctx context.Context, vppConn api.Connection, isIPv6 bool) (uint32, error) {
	now := time.Now()
	reply, err := ip.NewServiceClient(vppConn).IPTableAllocate(ctx, &ip.IPTableAllocate{
		Table: ip.IPTable{
			TableID: ^uint32(0),
			IsIP6:   isIPv6,
		},
	})
	if err != nil {
		return ^uint32(0), errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("vrfID", reply.Table.TableID).
		WithField("isIP6", isIPv6).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPTableAllocate").Debug("completed")
	return reply.Table.TableID, nil
}

func attach(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex, isClient bool) error  {
	for _, isIPv6 := range[]bool{false, true} {
		now := time.Now()
		vrfId, ok := Load(ctx, isClient, isIPv6)
		if !ok {
			/* Use default vrf ID*/
			vrfId = 0
		}
		if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetTable(ctx, &interfaces.SwInterfaceSetTable{
			SwIfIndex: swIfIndex,
			IsIPv6:    isIPv6,
			VrfID:     vrfId,
		}); err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("swIfIndex", swIfIndex).
			WithField("isIPv6", isIPv6).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceSetTable").Debug("completed")
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, t *vrfMap, isIPv6, isClient bool) {
	if vrfID, ok := LoadAndDelete(ctx, isClient, isIPv6); ok {
		t.mut.Lock()
		t.entries[conn.NetworkService].count--

		/* If there are no more clients using the vrf - delete it */
		if t.entries[conn.NetworkService].count == 0 {
			delete(t.entries, conn.NetworkService)
			_ = delVPP(ctx, vppConn, vrfID, isIPv6)
		}
		t.mut.Unlock()
	}
}

func delVPP(ctx context.Context, vppConn api.Connection, vrfID uint32, isIPv6 bool) error {
	now := time.Now()
	_, err := ip.NewServiceClient(vppConn).IPTableAddDel(ctx, &ip.IPTableAddDel{
		IsAdd: false,
		Table: ip.IPTable{
			TableID: vrfID,
			IsIP6:   isIPv6,
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("isAdd", false).
		WithField("vrfID", vrfID).
		WithField("isIP6", isIPv6).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPTableAddDel").Debug("completed")
	return nil
}
