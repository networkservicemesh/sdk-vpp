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

package vrf

import (
	"sync"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type vrfInfo struct {
	/* vrf ID */
	id uint32

	/* count - the number of clients using this vrf ID */
	count uint32
}

type vrfMap struct {
	/* entries - is a map[NetworkServiceName]{vrfId, count} */
	entries map[string]*vrfInfo

	/* mutex for entries */
	mut sync.Mutex
}

func newMap() *vrfMap {
	return &vrfMap{
		entries: make(map[string]*vrfInfo),
	}
}

type options struct {
	loadFn ifindex.LoadInterfaceFn
}

// Option is an option pattern for upClient/Server
type Option func(o *options)

// WithLoadInterfaceFn allows to attach rtf table to custom interface.
// This option might be useful for example, for a loopback interface.
func WithLoadInterfaceFn(loadFn ifindex.LoadInterfaceFn) Option {
	return func(o *options) {
		o.loadFn = loadFn
	}
}
