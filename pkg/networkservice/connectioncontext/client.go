// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

package connectioncontext

import (
	"git.fd.io/govpp.git/api"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/routes"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/mtu"
)

// NewClient creates a NetworkServiceClient chain element to set the ip address on a vpp interface
// It applies the connection context to the *vpp* side of an interface leaving the
// Endpoint.
//                                               Endpoint
//                              +-------------------------------------+
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |        networkservice.NewClient()+-------------------+
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              |                                     |
//                              +-------------------------------------+
//
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		mtu.NewClient(vppConn),
		routes.NewClient(vppConn),
		ipaddress.NewClient(vppConn),
	)
}
