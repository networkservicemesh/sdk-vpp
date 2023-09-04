// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
	"time"

	"github.com/pkg/errors"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func create(ctx context.Context, vppConn api.Connection, networkService string, t *vrfMap, isIPv6 bool) (vtfID uint32, err error) {
	t.mut.Lock()
	defer t.mut.Unlock()

	info, contains := t.entries[networkService]
	if !contains {
		vrfID, err := createVPP(ctx, vppConn, isIPv6)
		if err != nil {
			return vrfID, err
		}
		info = &vrfInfo{
			id:       vrfID,
			attached: make(map[interface_types.InterfaceIndex]struct{}),
		}
		t.entries[networkService] = info
	}

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
		return ^uint32(0), errors.Wrap(err, "vppapi IPTableAllocate returned error")
	}
	log.FromContext(ctx).
		WithField("vrfID", reply.Table.TableID).
		WithField("isIP6", isIPv6).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPTableAllocate").Debug("completed")
	return reply.Table.TableID, nil
}

func attach(ctx context.Context, vppConn api.Connection, networkService string, m *Map, swIfIndex interface_types.InterfaceIndex, isClient bool) error {
	for _, isIPv6 := range []bool{false, true} {
		t := m.ipv4
		if isIPv6 {
			t = m.ipv6
		}

		t.mut.Lock()
		defer t.mut.Unlock()
		vrfInfo := t.entries[networkService]
		_, attached := vrfInfo.attached[swIfIndex]

		if !attached {
			now := time.Now()
			vrfID, ok := Load(ctx, isClient, isIPv6)
			if !ok {
				/* Use default vrf ID*/
				vrfID = 0
			}
			if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetTable(ctx, &interfaces.SwInterfaceSetTable{
				SwIfIndex: swIfIndex,
				IsIPv6:    isIPv6,
				VrfID:     vrfID,
			}); err != nil {
				return errors.Wrap(err, "vppapi SwInterfaceSetTable returned error")
			}

			log.FromContext(ctx).
				WithField("swIfIndex", swIfIndex).
				WithField("isIPv6", isIPv6).
				WithField("duration", time.Since(now)).
				WithField("vppapi", "SwInterfaceSetTable").Debug("completed")

			t.entries[networkService].attached[swIfIndex] = struct{}{}
		}
	}
	return nil
}

func del(ctx context.Context, vppConn api.Connection, networkService string, t *vrfMap, isIPv6, isClient bool) {
	if vrfID, ok := Load(ctx, isClient, isIPv6); ok {
		t.mut.Lock()
		if vrfInfo, ok := t.entries[networkService]; ok {
			swIfIndex, _ := ifindex.Load(ctx, isClient)
			delete(vrfInfo.attached, swIfIndex)
			log.FromContext(ctx).
				WithField("swIfIndex", swIfIndex).
				WithField("networkService", networkService).
				Debugf("swIfIndex deleted from vrfInfo.attached map")

			/* If there are no more clients using the vrf - delete it */
			if len(vrfInfo.attached) == 1 {
				delete(t.entries, networkService)
				_ = delVPP(ctx, vppConn, vrfID, isIPv6)
			}
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
		return errors.Wrap(err, "vppapi IPTableAddDel returned error")
	}
	log.FromContext(ctx).
		WithField("isAdd", false).
		WithField("vrfID", vrfID).
		WithField("isIP6", isIPv6).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IPTableAddDel").Debug("completed")
	return nil
}
func delV46(ctx context.Context, vppConn api.Connection, m *Map, networkService string, isClient bool) {
	del(ctx, vppConn, networkService, m.ipv6, true, isClient)
	del(ctx, vppConn, networkService, m.ipv4, false, isClient)
}

func delTableFromMetadataV46(ctx context.Context, isClient bool) {
	Delete(ctx, isClient, true)
	Delete(ctx, isClient, false)
}
