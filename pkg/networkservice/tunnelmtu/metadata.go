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

package tunnelmtu

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// Store sets the tunnel MTU stored in per Connection.Id metadata.
func Store(ctx context.Context, isClient bool, mtu uint32) {
	metadata.Map(ctx, isClient).Store(key{}, mtu)
}

// Delete deletes the tunnel MTU stored in per Connection.Id metadata
func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// Load returns the tunnel MTU stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context, isClient bool) (value uint32, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(uint32)
	return value, ok
}
