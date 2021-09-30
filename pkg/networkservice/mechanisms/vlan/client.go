// Copyright (c) 2021 Nordix Foundation.
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

package vlan

import (
	"context"

	"git.fd.io/govpp.git/api"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vlan/hwaddress"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vlan/l2vtr"
)

const (
	viaLabel = "via"
)

type vlanClient struct {
	vppConn     api.Connection
	deviceNames map[string]string
}

// NewClient returns a VLAN client chain element
func NewClient(vppConn api.Connection, domain2Device map[string]string) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		hwaddress.NewClient(vppConn),
		l2vtr.NewClient(vppConn),
		&vlanClient{
			vppConn:     vppConn,
			deviceNames: domain2Device,
		},
	)
}

func (v *vlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	mechanism := &networkservice.Mechanism{
		Cls:        cls.REMOTE,
		Type:       vlanmech.MECHANISM,
		Parameters: make(map[string]string),
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := addSubIf(ctx, conn, v.vppConn, v.deviceNames); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := v.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (v *vlanClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_ = delSubIf(ctx, conn, v.vppConn)
	return next.Client(ctx).Close(ctx, conn, opts...)
}
