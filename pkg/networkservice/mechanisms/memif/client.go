// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// +build !windows

package memif

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif/memifproxy"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type memifClient struct {
	vppConn api.Connection
}

// NewClient provides a NetworkServiceClient chain elements that support the memif Mechanism
func NewClient(vppConn api.Connection) networkservice.NetworkServiceClient {
	m := &memifClient{
		vppConn: vppConn,
	}

	return chain.NewNetworkServiceClient(
		m,
		switchcase.NewClient(&switchcase.ClientCase{
			Condition: func(ctx context.Context, conn *networkservice.Connection) bool {
				_, ok := loadDirectMemifInfo(ctx)
				return !ok
			},
			Client: memifproxy.New(),
		}),
	)
}

func mechanismsContain(list []*networkservice.Mechanism, t string) bool {
	for _, m := range list {
		if m.Type == t {
			return true
		}
	}
	return false
}

func (m *memifClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if !mechanismsContain(request.MechanismPreferences, memif.MECHANISM) {
		request.MechanismPreferences = append(request.MechanismPreferences, &networkservice.Mechanism{
			Cls:        cls.LOCAL,
			Type:       memif.MECHANISM,
			Parameters: make(map[string]string),
		})
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// if direct memif enabled save socket filename to metadata
	_, ok := loadDirectMemifInfo(ctx)
	if mechanism := memif.ToMechanism(conn.GetMechanism()); mechanism != nil && ok {
		storeDirectMemifInfo(ctx, directMemifInfo{socketURL: mechanism.GetSocketFileURL()})
	} else if err := create(ctx, conn, m.vppConn, metadata.IsClient(m)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := m.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (m *memifClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_ = del(ctx, conn, m.vppConn, metadata.IsClient(m))
	return next.Client(ctx).Close(ctx, conn, opts...)
}
