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

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/mtu"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/connectioncontext/ipcontext/routes"
)

// NewServer creates a NetworkServiceServer chain element to set the ip address on a vpp interface
// It applies the connection context to the *vpp* side of an interface plugged into the
// Endpoint.
//                                         Endpoint
//                              +------------------------------------+
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//          +-------------------+ networkservice.NewServer()      |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              |                                    |
//                              +------------------------------------+
//
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		mtu.NewServer(vppConn),
		routes.NewServer(vppConn),
		ipaddress.NewServer(vppConn),
	)
}
