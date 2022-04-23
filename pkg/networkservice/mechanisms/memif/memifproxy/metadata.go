// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
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

//go:build linux
// +build linux

package memifproxy

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

func store(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, false).Store(key{}, cancel)
}

func load(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}

func loadAndDelete(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}

type infoKey struct{}

// Info contains client NSURL and SocketFile needed for direct memif
type Info struct {
	NSURL, SocketFile string
}

func storeInfo(ctx context.Context, val *Info) {
	metadata.Map(ctx, true).Store(infoKey{}, val)
}

// LoadInfo loads Info stored in context in metadata
func LoadInfo(ctx context.Context) (value *Info, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).Load(infoKey{})
	if !ok {
		return
	}
	value, ok = rawValue.(*Info)
	return value, ok
}
