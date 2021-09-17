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
	"context"

	"github.com/edwarnicke/govpp/binapi/interface_types"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// Store sets the loopback swIfIndex stored in per Connection.Id metadata.
func Store(ctx context.Context, isClient bool, swIfIndex interface_types.InterfaceIndex) {
	metadata.Map(ctx, isClient).Store(key{}, swIfIndex)
}

// Delete deletes the swIfIndex stored in per Connection.Id metadata
func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// Load returns the swIfIndex stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(interface_types.InterfaceIndex)
	return value, ok
}

// LoadOrStore returns the existing swIfIndex stored in per Connection.Id metadata if present.
// Otherwise, it stores and returns the given swIfIndex.
// The loaded result is true if the value was loaded, false if stored.
func LoadOrStore(ctx context.Context, isClient bool, swIfIndex interface_types.InterfaceIndex) (value interface_types.InterfaceIndex, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{}, swIfIndex)
	if !ok {
		return
	}
	value, ok = rawValue.(interface_types.InterfaceIndex)
	return value, ok
}

// LoadAndDelete deletes the swIfIndex stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func LoadAndDelete(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(interface_types.InterfaceIndex)
	return value, ok
}
