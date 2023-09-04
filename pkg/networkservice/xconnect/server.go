// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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

package xconnect

import (
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect/l2xconnect"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect/l3xconnect"
)

// NewServer - creates new xconnect server chain element to that correctly handles payload.IP and payload.Ethernet
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		l2xconnect.NewServer(vppConn),
		l3xconnect.NewServer(vppConn),
	)
}
