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

package unnumbered

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type unnumberedServer struct {
	vppConn api.Connection
}

func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &unnumberedServer{
		vppConn: vppConn,
	}
}

func (u *unnumberedServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if _, ok := load(ctx, metadata.IsClient(u)); !ok {
		if err := addDel(ctx, u.vppConn, metadata.IsClient(u),true); err != nil {
			return nil, err
		}
		/* Unnumbered IPv6 interface doesn't work without IPv6 enabling */
		if err := enableIp6(ctx, u.vppConn, metadata.IsClient(u)); err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := u.Close(closeCtx, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}
			return nil, err
		}
		store(ctx, metadata.IsClient(u))
	}

	return conn, nil
}

func (u *unnumberedServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delete(ctx, metadata.IsClient(u))
	return next.Server(ctx).Close(ctx, conn)
}
