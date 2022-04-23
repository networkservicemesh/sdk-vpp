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

//go:build linux
// +build linux

package forwarder

import (
	"net/url"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/cleanup"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/stats"
)

type forwarderOptions struct {
	name            string
	authorizeServer networkservice.NetworkServiceServer
	clientURL       *url.URL
	dialTimeout     time.Duration
	domain2Device   map[string]string
	statsOpts       []stats.Option
	cleanupOpts     []cleanup.Option
	vxlanOpts       []vxlan.Option
	dialOpts        []grpc.DialOption
}

// Option is an option pattern for forwarder chain elements
type Option func(o *forwarderOptions)

// WithName - set a forwarder name
func WithName(name string) Option {
	return func(o *forwarderOptions) {
		o.name = name
	}
}

// WithAuthorizeServer sets authorization server chain element
func WithAuthorizeServer(authorizeServer networkservice.NetworkServiceServer) Option {
	if authorizeServer == nil {
		panic("Authorize server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.authorizeServer = authorizeServer
	}
}

// WithClientURL sets clientURL.
func WithClientURL(clientURL *url.URL) Option {
	return func(c *forwarderOptions) {
		c.clientURL = clientURL
	}
}

// WithDialTimeout sets dial timeout for the client
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *forwarderOptions) {
		o.dialTimeout = dialTimeout
	}
}

// WithVlanDomain2Device sets vlan option
func WithVlanDomain2Device(domain2Device map[string]string) Option {
	return func(o *forwarderOptions) {
		o.domain2Device = domain2Device
	}
}

// WithStatsOptions sets stats options
func WithStatsOptions(opts ...stats.Option) Option {
	return func(o *forwarderOptions) {
		o.statsOpts = opts
	}
}

// WithCleanupOptions sets cleanup options
func WithCleanupOptions(opts ...cleanup.Option) Option {
	return func(o *forwarderOptions) {
		o.cleanupOpts = opts
	}
}

// WithVxlanOptions sets vxlan options
func WithVxlanOptions(opts ...vxlan.Option) Option {
	return func(o *forwarderOptions) {
		o.vxlanOpts = opts
	}
}

// WithDialOptions sets dial options
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *forwarderOptions) {
		o.dialOpts = opts
	}
}
