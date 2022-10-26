// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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
	"strconv"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

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
		err = errors.WithStack(err)
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

func setConnContextMTU(conn *networkservice.Connection) {
	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetMTU() == 0 {
		conn.GetContext().MTU = jumboFrameSize
	}

	if mechMTU, err := fromMechanism(conn.GetMechanism()); err == nil && mechMTU < conn.GetContext().GetMTU() {
		conn.GetContext().MTU = mechMTU
	}
}

func fromMechanism(mechanism *networkservice.Mechanism) (val uint32, err error) {
	if mechanism.GetParameters() == nil {
		return 0, errors.New("mechanism is empty")
	}
	mtu, err := strconv.ParseUint(mechanism.GetParameters()[common.MTU], 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(mtu), nil
}
