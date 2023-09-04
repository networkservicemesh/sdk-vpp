// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

package tag

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return nil
	}

	now := time.Now()
	if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceTagAddDel(ctx, &interfaces.SwInterfaceTagAddDel{
		IsAdd:     true,
		SwIfIndex: swIfIndex,
		Tag:       conn.GetId(),
	}); err != nil {
		return errors.Wrap(err, "vppapi SwInterfaceTagAddDel returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("tag", conn.GetId()).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceTagAddDel").Debug("completed")
	return nil
}
