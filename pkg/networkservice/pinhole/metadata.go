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

package pinhole

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// StoreExtra sets an extra IPPort stored in per Connection.Id metadata.
func StoreExtra(ctx context.Context, isClient bool, ipPort *IPPort) {
	metadata.Map(ctx, isClient).Store(key{}, ipPort)
}

// DeleteExtra deletes an extra IPPort stored in per Connection.Id metadata
func DeleteExtra(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// LoadExtra returns an extra IPPort stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func LoadExtra(ctx context.Context, isClient bool) (value *IPPort, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(*IPPort)
	return value, ok
}
