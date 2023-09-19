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

package ipsec

import (
	"context"
	"crypto/rsa"
	"net"
	"sync"

	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipsecMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/ipsec"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/ipsec/mtu"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/pinhole"
)

type ipsecServer struct {
	vppConn  api.Connection
	tunnelIP net.IP

	privateKeyFileName string
	privateKey         *rsa.PrivateKey
	onceInit           sync.Once
}

// NewServer - returns a new server for the IPSec remote mechanism
func NewServer(vppConn api.Connection, tunnelIP net.IP, options ...Option) networkservice.NetworkServiceServer {
	opts := &ipsecOptions{}
	for _, opt := range options {
		opt(opts)
	}

	if opts.privateKeyFileName == "" {
		var err error
		opts.privateKeyFileName, err = GenerateRSAKey()
		if err != nil {
			log.FromContext(context.Background()).Fatalf("ipsecServer GenerateRSAKey error: %v", err)
		}
	}

	privateKey, err := privateKeyFromFile(opts.privateKeyFileName)
	if err != nil {
		log.FromContext(context.Background()).Fatalf("ipsecServer privateKeyFromFile error: %v", err)
	}

	return chain.NewNetworkServiceServer(
		mtu.NewServer(vppConn, tunnelIP),
		&ipsecServer{
			vppConn:            vppConn,
			tunnelIP:           tunnelIP,
			privateKeyFileName: opts.privateKeyFileName,
			privateKey:         privateKey,
		},
	)
}

func (i *ipsecServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.IP {
		return next.Server(ctx).Request(ctx, request)
	}
	var err error
	i.onceInit.Do(func() {
		err = setIKEv2LocalKey(ctx, i.vppConn, i.privateKeyFileName)
	})
	if err != nil {
		return nil, err
	}

	if mechanism := ipsecMech.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		mechanism.SetDstIP(i.tunnelIP)
		mechanism.SetDstPort(ikev2DefaultPort)

		// Store extra IPPort entry to allow IKE protocol - https://www.rfc-editor.org/rfc/rfc5996
		pinhole.StoreExtra(ctx, metadata.IsClient(i), pinhole.NewIPPort(i.tunnelIP.String(), 500))
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if mechanism := ipsecMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		certificate, err := createCertBase64(i.privateKey, metadata.IsClient(i))
		if err != nil {
			return nil, err
		}
		mechanism.SetDstPublicKey(certificate)

		err = create(ctx, conn, i.vppConn, metadata.IsClient(i))
		if err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := i.Close(closeCtx, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}

			return nil, err
		}
	}

	return conn, nil
}

func (i *ipsecServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if mechanism := ipsecMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		delInterface(ctx, conn, i.vppConn, metadata.IsClient(i))
	}
	return next.Server(ctx).Close(ctx, conn)
}
