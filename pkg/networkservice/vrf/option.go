// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/govpp/binapi/interface_types"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type vrfInfo struct {
	/* vrf ID */
	id uint32

	/* attached - attached interfaces */
	attached map[interface_types.InterfaceIndex]struct{}
}

type vrfMap struct {
	entries map[string]*vrfInfo
	mut     sync.Mutex
}

// Map contains ipv6 and ipv4 vrf entries.
type Map struct {
	ipv6 *vrfMap
	ipv4 *vrfMap
}

// NewMap creates a new vrf.Map that can be used together with client and server.
func NewMap() *Map {
	return &Map{
		ipv6: &vrfMap{
			entries: make(map[string]*vrfInfo),
		},
		ipv4: &vrfMap{
			entries: make(map[string]*vrfInfo),
		},
	}
}

type options struct {
	m      *Map
	loadFn ifindex.LoadInterfaceFn
}

// Option is an option pattern for upClient/Server
type Option func(o *options)

// WithSharedMap - sets shared vrfV4 and vrfV6 map.
func WithSharedMap(v *Map) Option {
	return func(o *options) {
		o.m = v
	}
}

// WithLoadInterface replaces for for loading iface to attach the vrf table.
func WithLoadInterface(loadFn ifindex.LoadInterfaceFn) Option {
	return func(o *options) {
		o.loadFn = loadFn
	}
}
