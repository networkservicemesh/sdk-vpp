// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
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

package vxlan

import (
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
)

// Option is an option pattern for vxlan server/client
type Option func(o *vxlanOptions)

// WithPort sets vxlan udp port
func WithPort(port uint16) Option {
	return func(o *vxlanOptions) {
		if port != 0 {
			o.vxlanPort = port
		}
	}
}

// WithDump - sets dump parameters
func WithDump(dump *dumptool.DumpOption) Option {
	return func(o *vxlanOptions) {
		o.dumpOpt = dump
	}
}


type vxlanOptions struct {
	vxlanPort uint16
	vniKeys   []*vni.VniKey
	dumpOpt *dumptool.DumpOption
}
