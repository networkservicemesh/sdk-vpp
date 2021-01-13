// Copyright (c) 2020 Cisco and/or its affiliates.
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

package vxlanacl

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type vxlanACLServer struct {
	vppConn api.Connection
	IPMap
}

// NewServer - returns a new client that will set an ACL permitting vxlan packets through if and only if there's an ACL on the interface
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &vxlanACLServer{
		vppConn: vppConn,
	}
}

func (v *vxlanACLServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := vxlan.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		if _, ok := v.IPMap.LoadOrStore(mechanism.DstIP().String(), struct{}{}); !ok {
			if err := create(ctx, v.vppConn, mechanism.DstIP(), aclTag); err != nil {
				return nil, err
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (v *vxlanACLServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
