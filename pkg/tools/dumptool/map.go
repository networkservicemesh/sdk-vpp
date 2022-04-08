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

package dumptool

import (
	"context"
	"time"

	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

// Map -
type Map struct {
	innerMap
	ctx     context.Context
	timeout time.Duration
}

// NewMap -
func NewMap(ctx context.Context, timeout time.Duration) *Map {
	return &Map{
		ctx:     ctx,
		timeout: timeout,
	}
}

// Store -
func (m *Map) Store(connID string, val interface{}, onDelete DeleteFn) {
	m.innerMap.Store(connID, val)

	if onDelete !=nil {
		timeClock := clock.FromContext(m.ctx)
		expireCh := timeClock.After(m.timeout)
		go func() {
			<-expireCh
			if val, loaded := m.innerMap.LoadAndDelete(connID); loaded {
				_ = onDelete(val.(interface_types.InterfaceIndex))
			}
		}()
	}
}

// LoadOrStore -
func (m *Map) LoadOrStore(connID string, val interface{}, onDelete DeleteFn) (interface{}, bool) {
	v, loaded := m.innerMap.LoadOrStore(connID, val)

	if !loaded && onDelete !=nil {
		timeClock := clock.FromContext(m.ctx)
		expireCh := timeClock.After(m.timeout)
		go func() {
			<-expireCh
			if val, loaded := m.innerMap.LoadAndDelete(connID); loaded {
				_ = onDelete(val.(interface_types.InterfaceIndex))
			}
		}()
	}
	return v, loaded
}
