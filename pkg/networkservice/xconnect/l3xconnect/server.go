// Copyright (c) 2021 Cisco and/or its affiliates.
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

package l3xconnect

import (
	"context"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type l3XconnectServer struct {
	vppConn api.Connection
	dumpMap *dumptool.Map
}

// NewServer returns a Server chain element that will cross connect a client and server vpp interface (if present)
func NewServer(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceServer {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	ctx := context.Background()
	dumpMap := dumptool.NewMap(ctx, 0)
	if o.dumpOpt != nil {
		var err error
		dumpMap, err = dump(ctx, vppConn, o.dumpOpt.PodName, o.dumpOpt.Timeout, false)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to Dump: %v", err)
			/* TODO: set empty dumpMap here? */
		}
	}

	return &l3XconnectServer{
		vppConn: vppConn,
		dumpMap: dumpMap,
	}
}

func (v *l3XconnectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.IP {
		return next.Server(ctx).Request(ctx, request)
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	_,_ = v.dumpMap.LoadAndDelete(conn.GetId())
	if err := create(ctx, v.vppConn, conn); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := v.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (v *l3XconnectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetPayload() != payload.IP {
		return next.Server(ctx).Close(ctx, conn)
	}
	_,_ = v.dumpMap.LoadAndDelete(conn.GetId())
	_ = del(ctx, v.vppConn)
	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	return rv, nil
}
