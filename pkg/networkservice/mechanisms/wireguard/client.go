// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco and/or its affiliates.
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

package wireguard

import (
	"context"
	"net"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	wireguardMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/wireguard"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/wireguard/mtu"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/wireguard/peer"
)

type wireguardClient struct {
	vppConn  api.Connection
	tunnelIP net.IP
}

// NewClient - returns a new client for the wireguard remote mechanism
func NewClient(vppConn api.Connection, tunnelIP net.IP) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		peer.NewClient(vppConn),
		&wireguardClient{
			vppConn:  vppConn,
			tunnelIP: tunnelIP,
		},
		mtu.NewClient(vppConn, tunnelIP),
	)
}

func (w *wireguardClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.IP {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	privateKey, _ := wgtypes.GeneratePrivateKey()
	publicKey := privateKey.PublicKey().String()
	// If we already have a key we can reuse it
	// else create new key and store it after successful interface creation
	if mechanism := wireguardMech.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		// If there is a key in mechanism then we can use it
		publicKey = mechanism.SrcPublicKey()
	}
	mechanism := &networkservice.Mechanism{
		Cls:        cls.REMOTE,
		Type:       MECHANISM,
		Parameters: make(map[string]string),
	}
	wireguardMech.ToMechanism(mechanism).
		SetSrcPublicKey(publicKey).
		SetSrcIP(w.tunnelIP).
		SetSrcPort(wireguardDefaultPort)

	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if _, err = createInterface(ctx, conn, w.vppConn, privateKey, metadata.IsClient(w)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := w.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (w *wireguardClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if conn.GetPayload() != payload.IP {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}
	_ = delInterface(ctx, conn, w.vppConn, metadata.IsClient(w))
	return next.Client(ctx).Close(ctx, conn, opts...)
}
