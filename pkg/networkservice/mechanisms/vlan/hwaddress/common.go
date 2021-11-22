// Copyright (c) 2021 Nordix Foundation.
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

package hwaddress

import (
	"context"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/pkg/errors"

	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

func setEthContextHwaddress(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		now := time.Now()
		swIfIndex, ok := ifindex.Load(ctx, isClient)
		if !ok {
			return nil
		}

		rsp, err := interfaces.NewServiceClient(vppConn).SwInterfaceGetMacAddress(ctx, &interfaces.SwInterfaceGetMacAddress{
			SwIfIndex: swIfIndex})
		if err != nil {
			return errors.WithStack(err)
		}
		log.FromContext(ctx).
			WithField("duration", time.Since(now)).
			WithField("HostInterfaceIndex", swIfIndex).
			WithField("HwAddress", rsp.MacAddress).
			WithField("vppapi", "SwInterfaceGetMacAddress").Debug("completed")

		if conn.GetContext().GetEthernetContext() == nil {
			conn.GetContext().EthernetContext = new(networkservice.EthernetContext)
		}
		ethernetContext := conn.GetContext().GetEthernetContext()
		if isClient {
			ethernetContext.SrcMac = rsp.MacAddress.String()
		} else {
			ethernetContext.DstMac = rsp.MacAddress.String()
		}
	}
	return nil
}
