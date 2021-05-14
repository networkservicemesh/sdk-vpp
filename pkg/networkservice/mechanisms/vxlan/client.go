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

package vxlan

import (
	"context"
	"net"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan/vxlanacl"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan/mtu"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan/vni"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type vxlanClient struct {
	vppConn api.Connection
}

// NewClient - returns a new client for the vxlan remote mechanism
func NewClient(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		&vxlanClient{
			vppConn: vppConn,
		},
		vxlanacl.NewClient(vppConn),
		mtu.NewClient(vppConn, tunnelIP),
		vni.NewClient(tunnelIP),
	)
}

func (v *vxlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.Ethernet {
		return next.Client(ctx).Request(ctx, request, opts...)
	}
	mechanism := &networkservice.Mechanism{
		Cls:        cls.REMOTE,
		Type:       MECHANISM,
		Parameters: make(map[string]string),
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := addDel(ctx, conn, v.vppConn, true, metadata.IsClient(v)); err != nil {
		_, _ = v.Close(ctx, conn, opts...)
		return nil, err
	}
	return conn, nil
}

func (v *vxlanClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if conn.GetPayload() != payload.Ethernet {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	if err := addDel(ctx, conn, v.vppConn, false, metadata.IsClient(v)); err != nil {
		return nil, err
	}
	return rv, nil
}
