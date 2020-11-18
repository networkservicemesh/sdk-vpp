// Copyright (c) 2020 Cisco and/or its affiliates.
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

package memif

import (
	"context"

	"github.com/edwarnicke/govpp/binapi/memif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

func store(ctx context.Context, isClient bool, socket *memif.MemifSocketFilenameAddDel) {
	metadata.Map(ctx, isClient).Store(key{}, socket)
}

func dElete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

func load(ctx context.Context, isClient bool) (value *memif.MemifSocketFilenameAddDel, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(*memif.MemifSocketFilenameAddDel)
	return value, ok
}
