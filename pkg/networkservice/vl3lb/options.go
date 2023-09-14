// Copyright (c) 2023 Cisco and/or its affiliates.
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

package vl3lb

import (
	"net/url"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/govpp/binapi/ip_types"
)

type vl3LBOptions struct {
	port       uint16
	targetPort uint16
	protocol   ip_types.IPProto
	selector   map[string]string

	clientURL   *url.URL
	dialTimeout time.Duration
	dialOpts    []grpc.DialOption
}

// Option is an option pattern for forwarder chain elements
type Option func(o *vl3LBOptions)

// WithPort - set a load balancer port
func WithPort(port uint16) Option {
	return func(o *vl3LBOptions) {
		o.port = port
		if o.targetPort == 0 {
			o.targetPort = port
		}
	}
}

// WithTargetPort - set a real server target port
func WithTargetPort(targetPort uint16) Option {
	return func(o *vl3LBOptions) {
		o.targetPort = targetPort
	}
}

// WithProtocol - set IP protocol
func WithProtocol(protocol ip_types.IPProto) Option {
	return func(o *vl3LBOptions) {
		o.protocol = protocol
	}
}

// WithSelector - set a load balancer selector
func WithSelector(selector map[string]string) Option {
	return func(o *vl3LBOptions) {
		o.selector = selector
	}
}

// WithClientURL sets clientURL.
func WithClientURL(clientURL *url.URL) Option {
	return func(c *vl3LBOptions) {
		c.clientURL = clientURL
	}
}

// WithDialTimeout sets dial timeout for the client
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *vl3LBOptions) {
		o.dialTimeout = dialTimeout
	}
}

// WithDialOptions sets dial options
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *vl3LBOptions) {
		o.dialOpts = opts
	}
}
