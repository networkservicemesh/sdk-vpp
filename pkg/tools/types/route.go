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
	"github.com/edwarnicke/govpp/binapi/fib_types"
)

// IsV6toFibProto - returns fib_types.FIB_API_PATH_NH_PROTO_IP6 if isv6 is true
//                  fib_types.FIB_API_PATH_NH_PROTO_IP4 if isv6 is false
// Example:
//    Given a *net.IPNet dst:
//    types.IsV6toFibProto(dst.IP.To4() == nil)
func IsV6toFibProto(isv6 bool) fib_types.FibPathNhProto {
	if isv6 {
		return fib_types.FIB_API_PATH_NH_PROTO_IP6
	}
	return fib_types.FIB_API_PATH_NH_PROTO_IP4
}
