// Copyright (c) 2020 Cisco and/or its affiliates.
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

package ipaddress

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

func addDel(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient, isAdd bool) error {
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return errors.New("no swIfIndex available")
	}
	ipNet := conn.GetContext().GetIpContext().GetDstIPNet()
	if isClient {
		ipNet = conn.GetContext().GetIpContext().GetSrcIPNet()
	}
	if ipNet == nil {
		return nil
	}
	now := time.Now()
	if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceAddDelAddress(ctx, &interfaces.SwInterfaceAddDelAddress{
		SwIfIndex: swIfIndex,
		IsAdd:     isAdd,
		Prefix:    types.ToVppAddressWithPrefix(ipNet),
	}); err != nil {
		return errors.WithStack(err)
	}
	logger.Log(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("prefix", ipNet).
		WithField("isAdd", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceAddDelAddress").Debug("completed")
	return nil
}
