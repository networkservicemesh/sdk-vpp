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

// +build linux

package kernel

import (
	"context"
	"os"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel/kernelvethpair"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/kernel/kerneltap"
)

type kernelClient struct{}

var _ networkservice.NetworkServiceClient = &kernelClient{}

// NewClient - returns a new Client chain element implementing the kernel mechanism with vpp
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	if _, err := os.Stat(vnetFilename); err == nil {
		return chain.NewNetworkServiceClient(
			&kernelClient{},
			kerneltap.NewClient(vppConn),
		)
	}
	return chain.NewNetworkServiceClient(
		&kernelClient{},
		kernelvethpair.NewClient(vppConn),
	)
}

func (k *kernelClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	mechanism := &networkservice.Mechanism{
		Cls:        cls.LOCAL,
		Type:       MECHANISM,
		Parameters: make(map[string]string),
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if err := create(ctx, conn, metadata.IsClient(k)); err != nil {
		_, _ = k.Close(ctx, conn, opts...)
		return nil, err
	}
	return conn, nil
}

func (k *kernelClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	if err := del(ctx, conn, metadata.IsClient(k)); err != nil {
		return nil, err
	}
	return rv, nil
}
