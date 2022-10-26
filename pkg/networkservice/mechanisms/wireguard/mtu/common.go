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
	// https://lists.zx2c4.com/pipermail/wireguard/2017-December/002201.html
	if !isV6 {
		//  20-byte outer IPv4 header
		//  8-byte outer UDP header
		//  4-byte type
		//  4-byte key index
		//  8-byte nonce
		// 16-byte authentication tag
		// 60 byte total
		return 60
	}
	//  40-byte outer IPv4 header
	//  8-byte outer UDP header
	//  4-byte type
	//  4-byte key index
	//  8-byte nonce
	// 16-byte authentication tag
	// 80 bytes total
	return 80
}
