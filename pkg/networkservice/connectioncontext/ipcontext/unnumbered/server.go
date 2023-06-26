// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type unnumberedServer struct {
	vppConn     api.Connection
	loadIfaceFn func(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool)
}

// NewServer creates a new instance of unnumbered server
func NewServer(vppConn api.Connection, loadIfaceFn func(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool)) networkservice.NetworkServiceServer {
	return &unnumberedServer{
		vppConn:     vppConn,
		loadIfaceFn: loadIfaceFn,
	}
}

func (u *unnumberedServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := addDel(ctx, u.vppConn, metadata.IsClient(u), true, u.loadIfaceFn); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := u.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		return nil, err
	}

	return conn, nil
}

func (u *unnumberedServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delete(ctx, metadata.IsClient(u))
	return next.Server(ctx).Close(ctx, conn)
}
