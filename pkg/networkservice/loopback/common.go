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

package loopback

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

/* Create loopback interface and store it in metadata */
func createLoopback(ctx context.Context, vppConn api.Connection, networkService string, t *Map, isClient bool) (err error) {
	if _, ok := Load(ctx, isClient); !ok {
		<-t.exec.AsyncExec(func() {
			/* Check if we have already created loopback for a given NetworkService previously */
			info, ok := t.entries[networkService]
			if !ok {
				var swIfIndex interface_types.InterfaceIndex
				swIfIndex, err = createLoopbackVPP(ctx, vppConn)
				if err != nil {
					return
				}
				info = &loopInfo{
					swIfIndex: swIfIndex,
				}
				t.entries[networkService] = info
			}
			info.count++
			Store(ctx, isClient, info.swIfIndex)
		})
	}
	return
}

func createLoopbackVPP(ctx context.Context, vppConn api.Connection) (interface_types.InterfaceIndex, error) {
	now := time.Now()
	reply, err := interfaces.NewServiceClient(vppConn).CreateLoopback(ctx, &interfaces.CreateLoopback{})
	if err != nil {
		return interface_types.InterfaceIndex(^uint32(0)), errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", reply.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "CreateLoopback").Debug("completed")
	return reply.SwIfIndex, nil
}

func del(ctx context.Context, vppConn api.Connection, networkService string, t *Map, isClient bool) {
	if swIfIndex, ok := LoadAndDelete(ctx, isClient); ok {
		t.exec.AsyncExec(func() {
			t.entries[networkService].count--

			/* If there are no more clients using the loopback - delete it */
			if t.entries[networkService].count == 0 {
				delete(t.entries, networkService)
				if err := delVPP(ctx, vppConn, swIfIndex); err != nil {
					log.FromContext(ctx).Errorf("unable to delete loopback interface: %v", err)
				}
			}
		})
	}
}

func delVPP(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) error {
	now := time.Now()
	_, err := interfaces.NewServiceClient(vppConn).DeleteLoopback(ctx, &interfaces.DeleteLoopback{
		SwIfIndex: swIfIndex,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "DeleteLoopback").Debug("completed")
	return nil
}
