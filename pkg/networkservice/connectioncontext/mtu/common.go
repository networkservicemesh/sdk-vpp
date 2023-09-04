// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

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
	setMTU := &interfaces.SwInterfaceSetMtu{
		SwIfIndex: swIfIndex,
		Mtu: []uint32{
			conn.GetContext().GetMTU(),
			conn.GetContext().GetMTU(),
			conn.GetContext().GetMTU(),
			conn.GetContext().GetMTU(),
		},
	}
	_, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetMtu(ctx, setMTU)
	if err != nil {
		err = errors.Wrap(err, "vppapi SwInterfaceSetMtu returned error")
		log.FromContext(ctx).
			WithField("SwIfIndex", setMTU.SwIfIndex).
			WithField("MTU", setMTU.Mtu).
			WithField("duration", time.Since(now)).
			WithField("error", err).
			WithField("vppapi", "SwInterfaceSetMtu").Debug("error")
		return err
	}
	log.FromContext(ctx).
		WithField("SwIfIndex", setMTU.SwIfIndex).
		WithField("MTU", setMTU.Mtu).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceSetMtu").Debug("completed")
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
