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

package types

import (
	"net"

	"github.com/edwarnicke/govpp/binapi/ip_types"
)

// ToVppAddress - converts addr to ip_types.Address
func ToVppAddress(addr net.IP) ip_types.Address {
	a := ip_types.Address{}
	if addr.To4() == nil {
		a.Af = ip_types.ADDRESS_IP6
		ip := [16]uint8{}
		copy(ip[:], addr)
		a.Un = ip_types.AddressUnionIP6(ip)
	} else {
		a.Af = ip_types.ADDRESS_IP4
		ip := [4]uint8{}
		copy(ip[:], addr.To4())
		a.Un = ip_types.AddressUnionIP4(ip)
	}
	return a
}

// ToVppAddressWithPrefix - converts prefix to ip_types.AddressWithPrefix
func ToVppAddressWithPrefix(prefix *net.IPNet) ip_types.AddressWithPrefix {
	return ip_types.AddressWithPrefix(ToVppPrefix(prefix))
}

// ToVppPrefix - converts prefix to ip_types.Prefix
func ToVppPrefix(prefix *net.IPNet) ip_types.Prefix {
	length, _ := prefix.Mask.Size()
	r := ip_types.Prefix{
		Address: ToVppAddress(prefix.IP),
		Len:     uint8(length),
	}
	return r
}
