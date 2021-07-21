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

// +build linux

package kerneltap

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type kernelTapClient struct {
	vppConn api.Connection
}

// NewClient - return a new Client chain element implementing the kernel mechanism with vpp using tapv2
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	return &kernelTapClient{
		vppConn: vppConn,
	}
}

func (k *kernelTapClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
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
	if err := create(ctx, conn, k.vppConn, metadata.IsClient(k)); err != nil {
		_, _ = k.Close(ctx, conn, opts...)
		return nil, err
	}
	return conn, nil
}

func (k *kernelTapClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	err := del(ctx, conn, k.vppConn, metadata.IsClient(k))
	if err != nil {
		log.FromContext(ctx).Error(err)
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
