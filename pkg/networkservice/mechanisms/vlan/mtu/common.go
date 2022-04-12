// Copyright (c) 2022 Nordix Foundation.
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

package mtu

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/pkg/errors"

	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const l3MtuIndex = 0

func getL3MTU(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) (uint32, error) {
	now := time.Now()
	dc, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{
		SwIfIndex: swIfIndex,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "error attempting to get interface dump client to determine MTU for swIfIndex %d", swIfIndex)
	}
	defer func() { _ = dc.Close() }()

	details, err := dc.Recv()
	if err != nil {
		return 0, errors.Wrapf(err, "error attempting to get interface details to determine MTU for swIfIndex %d", swIfIndex)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("details.LinkMtu", details.LinkMtu).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceDump").Debug("completed")
	return details.Mtu[l3MtuIndex], nil
}
