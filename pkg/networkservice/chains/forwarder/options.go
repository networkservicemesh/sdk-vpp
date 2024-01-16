// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/metrics"
)

type forwarderOptions struct {
	name                             string
	authorizeServer                  networkservice.NetworkServiceServer
	authorizeMonitorConnectionServer networkservice.MonitorConnectionServer
	clientURL                        *url.URL
	dialTimeout                      time.Duration
	domain2Device                    map[string]string
	mechanismPrioriyList             []string
	statsOpts                        []metrics.Option
	cleanupOpts                      []cleanup.Option
	vxlanOpts                        []vxlan.Option
	dialOpts                         []grpc.DialOption
	clientAdditionalFunctionality    []networkservice.NetworkServiceClient
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

// WithAuthorizeMonitorConnectionServer sets authorization server chain element
func WithAuthorizeMonitorConnectionServer(authorizeMonitorConnectionServer networkservice.MonitorConnectionServer) Option {
	if authorizeMonitorConnectionServer == nil {
		panic("Authorize monitor server cannot be nil")
	}
	return func(o *forwarderOptions) {
		o.authorizeMonitorConnectionServer = authorizeMonitorConnectionServer
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

// WithMechanismPriority sets mechanismpriority option
func WithMechanismPriority(priorityList []string) Option {
	return func(o *forwarderOptions) {
		o.mechanismPrioriyList = priorityList
	}
}

// WithStatsOptions sets metrics options
func WithStatsOptions(opts ...metrics.Option) Option {
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

// WithClientAdditionalFunctionality sets client additional functionality
func WithClientAdditionalFunctionality(additionalFunctionality ...networkservice.NetworkServiceClient) Option {
	return func(o *forwarderOptions) {
		o.clientAdditionalFunctionality = additionalFunctionality
	}
}
