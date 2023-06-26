// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/govpp/binapi/ethernet_types"
)

// ToVppMacAddress - converts *net.HardwareAddr to ethernet_types.MacAddress
func ToVppMacAddress(hardwareAddr *net.HardwareAddr) ethernet_types.MacAddress {
	hwAddr := [6]uint8{}
	copy(hwAddr[:], *hardwareAddr)
	return ethernet_types.MacAddress(hwAddr)
}
