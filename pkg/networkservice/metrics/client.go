// Copyright (c) 2024 Cisco and/or its affiliates.
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

// Package metrics provides chain elements for retrieving metrics from vpp
package metrics

import (
	"context"

	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/metrics/ifacename"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/metrics/stats"
)

// NewClient provides a NetworkServiceClient chain elements that retrieves vpp interface metrics and names.
func NewClient(ctx context.Context, vppConn api.Connection, options ...Option) networkservice.NetworkServiceClient {
	opts := &metricsOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return chain.NewNetworkServiceClient(
		stats.NewClient(ctx, stats.WithSocket(opts.socket)),
		ifacename.NewClient(ctx, vppConn, ifacename.WithSocket(opts.socket)),
	)
}
