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
	"io"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/pkg/errors"

	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func getMTU(ctx context.Context, vppConn api.Connection, ifName string) (uint32, error) {
	now := time.Now()
	dc, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{
		NameFilterValid: true,
		NameFilter:      ifName,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get interface dump client to determine MTU on %s", ifName)
	}

	for {
		details, err := dc.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get interface details to determine MTU on %s", ifName)
		}

		if (ifName != details.InterfaceName) && (afPacketNamePrefix+ifName != details.InterfaceName) {
			log.FromContext(ctx).
				WithField("InterfaceName", details.InterfaceName).
				WithField("vppapi", "SwInterfaceDetails").Debug("skipped")
			continue
		}
		log.FromContext(ctx).
			WithField("InterfaceName", details.InterfaceName).
			WithField("L3 MTU", details.Mtu[l3MtuIndex]).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "SwInterfaceDump").Debug("completed")
		return details.Mtu[l3MtuIndex], nil
	}
	return 0, errors.New("unable to find interface in vpp")
}
