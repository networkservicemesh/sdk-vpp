// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

type memifOptions struct {
	directMemifEnabled bool
	changeNetNS        bool
}

// Option is an option for the connect server
type Option func(o *memifOptions)

// WithDirectMemif turns on direct memif logic
func WithDirectMemif() Option {
	return func(o *memifOptions) {
		o.directMemifEnabled = true
	}
}

// WithChangeNetNS sets if memif client/server should change net NS instead of using own one for creating socket
func WithChangeNetNS() Option {
	return func(o *memifOptions) {
		o.changeNetNS = true
	}
}
