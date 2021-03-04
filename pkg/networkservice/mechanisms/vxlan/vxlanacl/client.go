// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
)

type vxlanACLClient struct {
	vppConn api.Connection
	IPMap
}

// NewClient - returns a new client that will set an ACL permitting vxlan packets through if and only if there's an ACL on the interface
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	return &vxlanACLClient{
		vppConn: vppConn,
	}
}

func (v *vxlanACLClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if _, ok := v.IPMap.LoadOrStore(mechanism.SrcIP().String(), struct{}{}); !ok {
			if err := create(ctx, v.vppConn, mechanism.SrcIP(), aclTag); err != nil {
				return nil, err
			}
		}
	}
	return conn, nil
}

func (v *vxlanACLClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn)
}
