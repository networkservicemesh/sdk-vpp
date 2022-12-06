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

package pinhole

import (
	"context"
	"net"
	"strconv"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

// IPPort stores IP and port for an ACL rule
type IPPort struct {
	ip   string
	port uint16
}

// NewIPPort returns *IPPort entry
func NewIPPort(ip string, port uint16) *IPPort {
	return &IPPort{
		ip:   ip,
		port: port,
	}
}

func fromMechanism(mechanism *networkservice.Mechanism, isClient bool) *IPPort {
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
	portStr, ok := mechanism.GetParameters()[portKey]
	if !ok {
		return nil
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil
	}
	return NewIPPort(ipStr, uint16(port))
}

func fromContext(ctx context.Context, isClient bool) *IPPort {
	v, ok := LoadExtra(ctx, isClient)
	if !ok {
		return nil
	}
	return v
}

// IP - converts string to net.IP
func (i *IPPort) IP() net.IP {
	return net.ParseIP(i.ip)
}

// Port - returns port
func (i *IPPort) Port() uint16 {
	return i.port
}
