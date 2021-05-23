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

package pinhole

import (
	"net"
	"strconv"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

type ipPortKey struct {
	ip   string
	port uint16
}

func fromMechanism(mechanism *networkservice.Mechanism, isClient bool) *ipPortKey {
	rv := &ipPortKey{}
	if mechanism.GetParameters() == nil {
		return nil
	}
	ipKey := common.DstIP
	portKey := common.DstPort
	if isClient {
		ipKey = common.SrcIP
		portKey = common.SrcPort
	}
	ipStr, ok := mechanism.GetParameters()[ipKey]
	if !ok || ipStr == "" {
		return nil
	}
	rv.ip = ipStr
	portStr, ok := mechanism.GetParameters()[portKey]
	if !ok {
		return nil
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil
	}
	rv.port = uint16(port)
	return rv
}

func (i *ipPortKey) IP() net.IP {
	return net.ParseIP(i.ip)
}

func (i *ipPortKey) Port() uint16 {
	return i.port
}
