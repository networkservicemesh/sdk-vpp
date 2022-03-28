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

package nsmonitor

import (
	"context"
)

type clientOptions struct {
	supplyMonitor func(ctx context.Context) Monitor
}

// Option is an option pattern for netns monitor client
type Option func(*clientOptions)

// WithSupplyMonitor sets netns monitor initialization func
func WithSupplyMonitor(supplyMonitort func(ctx context.Context) Monitor) Option {
	return func(c *clientOptions) {
		c.supplyMonitor = supplyMonitort
	}
}
