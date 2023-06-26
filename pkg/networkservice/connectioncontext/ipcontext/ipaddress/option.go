// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
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

package ipaddress

import (
	"context"

	"github.com/networkservicemesh/govpp/binapi/interface_types"
)

type options struct {
	loadIfIndex ifIndexFunc
}

// Option is an option pattern for ipaddressClient/Server
type Option func(o *options)

// ifIndexFunc is a function to load the interface index
type ifIndexFunc func(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool)

// WithLoadSwIfIndex - sets function to load the interface index
func WithLoadSwIfIndex(f ifIndexFunc) Option {
	return func(o *options) {
		o.loadIfIndex = f
	}
}
