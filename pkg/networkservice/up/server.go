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

package up

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type upServer struct {
	ctx     context.Context
	vppConn Connection
	sync.Once
	initErr error
}

// NewServer provides a NetworkServiceServer chain elements that 'up's the swIfIndex
func NewServer(ctx context.Context, vppConn Connection) networkservice.NetworkServiceServer {
	return &upServer{
		ctx:     ctx,
		vppConn: vppConn,
	}
}

func (u *upServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := u.init(ctx); err != nil {
		return nil, err
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := up(ctx, u.vppConn, true); err != nil {
		if closeErr := u.closeOnFailure(postponeCtxFunc, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		log.FromContext(ctx).Errorf("SOME UGLY ERROR: %v", err.Error())
		return nil, err
	}

	if err := up(ctx, u.vppConn, false); err != nil {
		if closeErr := u.closeOnFailure(postponeCtxFunc, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		log.FromContext(ctx).Errorf("SOME UGLY ERROR: %v", err.Error())
		return nil, err
	}

	return conn, nil
}

func (u *upServer) closeOnFailure(postponeCtxFunc func() (context.Context, context.CancelFunc), conn *networkservice.Connection) error {
	closeCtx, cancelClose := postponeCtxFunc()
	defer cancelClose()

	_, err := u.Close(closeCtx, conn)

	return err
}

func (u *upServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func (u *upServer) init(ctx context.Context) error {
	u.Do(func() {
		u.initErr = initFunc(ctx, u.vppConn)
	})
	return u.initErr
}
