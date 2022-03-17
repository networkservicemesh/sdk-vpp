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

package wireguard

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// store sets the public key stored in per Connection.Id metadata.
func store(ctx context.Context, pubkey string, isClient bool) {
	metadata.Map(ctx, isClient).Store(key{}, pubkey)
}

// loadAndDelete deletes the public key stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDelete(ctx context.Context, isClient bool) (value string, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(string)
	return value, ok
}

// load returns the public key stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func load(ctx context.Context, isClient bool) (value string, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(string)
	return value, ok
}
