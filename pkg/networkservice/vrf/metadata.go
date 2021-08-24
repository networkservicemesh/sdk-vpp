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

package vrf

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{
	isIPv6 bool
}

// Store sets the vrfID stored in per Connection.Id metadata.
func Store(ctx context.Context, isClient, isIPv6 bool, vrfID uint32) {
	metadata.Map(ctx, isClient).Store(key{isIPv6}, vrfID)
}

// Delete deletes the vrfID stored in per Connection.Id metadata
func Delete(ctx context.Context, isClient, isIPv6 bool) {
	metadata.Map(ctx, isClient).Delete(key{isIPv6})
}

// Load returns the vrfID stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context, isClient, isIPv6 bool) (value uint32, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{isIPv6})
	if !ok {
		return
	}
	value, ok = rawValue.(uint32)
	return value, ok
}

// LoadOrStore returns the existing vrfID stored in per Connection.Id metadata if present.
// Otherwise, it stores and returns the given vrfID.
// The loaded result is true if the value was loaded, false if stored.
func LoadOrStore(ctx context.Context, isClient, isIPv6 bool, vrfID uint32) (value uint32, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{isIPv6}, vrfID)
	if !ok {
		return
	}
	value, ok = rawValue.(uint32)
	return value, ok
}

// LoadAndDelete deletes the vrfID stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func LoadAndDelete(ctx context.Context, isClient, isIPv6 bool) (value uint32, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{isIPv6})
	if !ok {
		return
	}
	value, ok = rawValue.(uint32)
	return value, ok
}
