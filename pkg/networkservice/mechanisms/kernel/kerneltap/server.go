// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// +build linux

package kerneltap

import (
	"context"

	"git.fd.io/govpp.git/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type kernelTapServer struct {
	vppConn api.Connection
	dumpMap *dumptool.Map
}

// NewServer - return a new Server chain element implementing the kernel mechanism with vpp using tapv2
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

	return &kernelTapServer{
		vppConn: vppConn,
		dumpMap: dumpMap,
	}
}

func (k *kernelTapServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := create(ctx, conn, k.vppConn, k.dumpMap, metadata.IsClient(k)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := k.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (k *kernelTapServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	err := del(ctx, conn, k.vppConn, k.dumpMap, metadata.IsClient(k))
	if err != nil {
		log.FromContext(ctx).Error(err)
	}
	return next.Server(ctx).Close(ctx, conn)
}
