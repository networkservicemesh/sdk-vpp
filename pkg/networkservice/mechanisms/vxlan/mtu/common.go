// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

func overhead(isV6 bool) uint32 {
	if !isV6 {
		// outer ipv4 header - 20 bytes
		// outer udp header - 8 bytes
		// vxlan header - 8 bytes
		// inner ethernet header - 14 bytes
		// optional overhead for 802.1q vlan tags - 4 bytes
		// total - 54 bytes
		return 54
	}
	// outer ipv6 header - 40 bytes
	// outer udp header - 8 bytes
	// vxlan header - 8 bytes
	// inner ethernet header - 14 bytes
	// optional overhead for 802.1q vlan tags - 4 bytes
	// total - 74 bytes
	return 74
}
