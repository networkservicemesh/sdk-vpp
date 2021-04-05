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

package mtu

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

const (
	jumboFrameSize = 9000
)

func setVPPMTU(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	now := time.Now()
	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok || conn.GetContext().GetMTU() == 0 {
		return nil
	}
	setMTU := &interfaces.HwInterfaceSetMtu{
		SwIfIndex: swIfIndex,
		Mtu:       uint16(conn.GetContext().GetMTU()),
	}
	_, err := interfaces.NewServiceClient(vppConn).HwInterfaceSetMtu(ctx, setMTU)
	if err != nil {
		return err
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", setMTU.SwIfIndex).
		WithField("MTU", setMTU.Mtu).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "HwInterfaceSetMtu").Debug("completed")
	return nil
}

func setConnContextMTU(request *networkservice.NetworkServiceRequest) {
	if request.GetConnection().GetContext().GetMTU() != 0 {
		return
	}
	if request.GetConnection() == nil {
		request.Connection = &networkservice.Connection{}
	}
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	request.GetConnection().GetContext().MTU = jumboFrameSize
}
