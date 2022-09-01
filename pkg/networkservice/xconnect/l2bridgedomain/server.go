// Copyright (c) 2022 Nordix Foundation.
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

package l2bridgedomain

import (
	"context"

	"git.fd.io/govpp.git/api"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vlan"
)

type l2BridgeDomainServer struct {
	vppConn api.Connection
	b       l2BridgeDomain
}

// NewServer returns a Client chain element that will add client and server vpp interface (if present) to a dridge domain
func NewServer(vppConn api.Connection) networkservice.NetworkServiceServer {
	return &l2BridgeDomainServer{
		vppConn: vppConn,
	}
}

func (v *l2BridgeDomainServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	vlanID, ok := vlan.Load(ctx, true)
	// return if the belonging remote mechanism not vlan mechanism
	if !ok || request.GetConnection().GetPayload() != payload.Ethernet {
		return next.Server(ctx).Request(ctx, request)
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := addBridgeDomain(ctx, v.vppConn, &v.b, vlanID); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := v.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (v *l2BridgeDomainServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	vlanID, ok := vlan.Load(ctx, true)
	if !ok || conn.GetPayload() != payload.Ethernet {
		return next.Server(ctx).Close(ctx, conn)
	}
	if err := delBridgeDomain(ctx, v.vppConn, &v.b, vlanID); err != nil {
		log.FromContext(ctx).WithField("l2BridgeDomain", "server").Error("delBridgeDomain", err)
	}
	return next.Server(ctx).Close(ctx, conn)
}
