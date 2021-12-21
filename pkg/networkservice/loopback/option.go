// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package loopback

import (
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/serialize"
)

type loopInfo struct {
	/* loopback swIfIndex */
	swIfIndex interface_types.InterfaceIndex

	/* count - the number of clients using this loopback */
	count uint32
}

// Map stores loopback swIfIndex by NetworkServiceName
type Map struct {
	/* entries - is a map[NetworkServiceName]{swIfIndex, count} */
	entries map[string]*loopInfo

	/* executor */
	exec serialize.Executor
}

// NewMap creates loopback map
func NewMap() *Map {
	return &Map{
		entries: make(map[string]*loopInfo),
	}
}

type options struct {
	loopbacks *Map
}

// Option is an option pattern for loopbackClient/Server
type Option func(o *options)

// WithSharedMap - sets shared loopback map. It may be needed for sharing Map between client and server
func WithSharedMap(l *Map) Option {
	return func(o *options) {
		o.loopbacks = l
	}
}
