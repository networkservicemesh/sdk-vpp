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

package kernel

import (
	"os"

	"git.fd.io/govpp.git/api"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel/kernelvethpair"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel/kerneltap"
)

// NewServer return a NetworkServiceServer chain element that correctly handles the kernel Mechanism
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	if _, err := os.Stat(vnetFilename); err == nil {
		return kerneltap.NewServer(vppConn)
	}
	return kernelvethpair.NewServer(vppConn)
}
