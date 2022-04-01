// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
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

package vxlan

import (
	"context"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/vxlan"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"
	"io"
	"net"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/vxlan/vni"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/vxlan/mtu"
)

type vxlanServer struct {
	vppConn api.Connection
	dumpMap *dumptool.Map
}

// NewServer - returns a new server for the vxlan remote mechanism
func NewServer(vppConn api.Connection, tunnelIP net.IP, options ...Option) networkservice.NetworkServiceServer {
	opts := &vxlanOptions{
		vxlanPort: vxlanDefaultPort,
	}
	for _, opt := range options {
		opt(opts)
	}

	ctx := context.Background()
	dumpMap := dumptool.NewMap(ctx, 0)
	dumpVNI := dumptool.NewMap(ctx, 0)
	if opts.dumpOpt != nil {
		var err error
		dumpMap, err = dump(ctx, vppConn, opts.dumpOpt.PodName, opts.dumpOpt.Timeout, false)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to Dump: %v", err)
			/* TODO: set empty dumpMap here? */
		}

		dumpVNI,_ = dumptool.DumpInterfaces(ctx, vppConn, opts.dumpOpt.PodName, opts.dumpOpt.Timeout, false,
			func(details *interfaces.SwInterfaceDetails) (interface{}, error) {
				if details.InterfaceDevType == dumptool.DevTypeVxlan {
					vxClient, err := vxlan.NewServiceClient(vppConn).VxlanTunnelV2Dump(ctx, &vxlan.VxlanTunnelV2Dump{
						SwIfIndex: details.SwIfIndex,
					})
					if err != nil {
						return nil, err
					}
					defer func() { _ = vxClient.Close() }()

					vxDetails, err := vxClient.Recv()
					if err == io.EOF || vxDetails == nil {
						return nil, nil
					}
					return vni.NewVniKey(vxDetails.SrcAddress.String(), vxDetails.Vni), nil
				}
				return nil, nil
			},
			nil,
		)
	}

	var vniList []*vni.VniKey
	dumpVNI.Range(func(key string, value interface{}) bool {
		vniList = append(vniList, value.(*vni.VniKey))
		return true
	})

	return chain.NewNetworkServiceServer(
		vni.NewServer(tunnelIP, vni.WithTunnelPort(opts.vxlanPort), vni.WithVNIKeys(vniList)),
		mtu.NewServer(vppConn, tunnelIP),
		&vxlanServer{
			vppConn: vppConn,
			dumpMap: dumpMap,
		},
	)
}

func (v *vxlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetPayload() != payload.Ethernet {
		return next.Server(ctx).Request(ctx, request)
	}
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := addDel(ctx, conn, v.vppConn, v.dumpMap, true, metadata.IsClient(v)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := v.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (v *vxlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetPayload() != payload.Ethernet {
		return next.Server(ctx).Close(ctx, conn)
	}
	if err := addDel(ctx, conn, v.vppConn, v.dumpMap,false, metadata.IsClient(v)); err != nil {
		log.FromContext(ctx).WithField("vxlan", "server").Errorf("error while deleting vxlan connection: %v", err.Error())
	}

	return next.Server(ctx).Close(ctx, conn)
}
