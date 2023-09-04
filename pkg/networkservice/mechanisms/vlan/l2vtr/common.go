// Copyright (c) 2021-2022 Nordix Foundation.
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

package l2vtr

import (
	"context"
	"time"

	"github.com/networkservicemesh/govpp/binapi/l2"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func enableVtr(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if mechanism.GetVlanID() == 0 {
			return nil
		}
		swIfIndex, ok := ifindex.Load(ctx, true)
		if !ok {
			return nil
		}
		now := time.Now()
		if _, err := l2.NewServiceClient(vppConn).L2InterfaceVlanTagRewrite(ctx, &l2.L2InterfaceVlanTagRewrite{
			SwIfIndex: swIfIndex,
			VtrOp:     L2VtrPop1,
			PushDot1q: 0,
			Tag1:      0,
			Tag2:      0,
		}); err != nil {
			return errors.Wrap(err, "vppapi L2InterfaceVlanTagRewrite returned error")
		}
		log.FromContext(ctx).
			WithField("duration", time.Since(now)).
			WithField("SwIfIndex", swIfIndex).
			WithField("operation", "POP 1").
			WithField("vppapi", "L2InterfaceVlanTagRewrite").Debug("completed")
	}
	return nil
}
