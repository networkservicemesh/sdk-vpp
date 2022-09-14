// Copyright (c) 2021-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

//go:build linux
// +build linux

package mtu

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type mtuClient struct{}

// NewClient provides a NetworkServiceClient that sets the MTU on a kernel interface
// It sets the MTU on the *kernel* side of an interface leaving the
// Client.  Generally only used by privileged Clients like those implementing
// the Cross Connect Network Service for K8s (formerly known as NSM Forwarder).
//
//	           Client
//	+---------------------------+
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           +-------------------+
//	|                           |          mtu.NewClient()
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	+---------------------------+
func NewClient() networkservice.NetworkServiceClient {
	return &mtuClient{}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("mtuClient", "Request")

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := setMTU(ctx, conn, metadata.IsClient(m)); err != nil {
		logger.Debugf("about to Close due to error: %s", err.Error())

		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := m.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
