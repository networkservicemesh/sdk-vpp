// Copyright (c) 2022 Cisco and/or its affiliates.
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

// +build linux

package kerneltap

import "github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"

type options struct {
	dump dumptool.DumpNSMFn
}

// Option is an option pattern for kernel
type Option func(o *options)

// WithDump - sets dump parameters
func WithDump(dump dumptool.DumpNSMFn) Option {
	return func(o *options) {
		o.dump = dump
	}
}
