// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

package pinhole

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type pinholeClient struct {
	vppConn api.Connection
	ipPortMap

	// We need to protect ACL rules applying with a mutex.
	// Because adding new entries is based on a dump and applying modified data.
	// This must be an atomic operation, otherwise a data race is possible.
	mutex *sync.Mutex
}

// NewClient - returns a new client that will set an ACL permitting remote protocols packets through if and only if there's an ACL on the interface
func NewClient(vppConn api.Connection, opts ...Option) networkservice.NetworkServiceClient {
	o := &option{
		mutex: new(sync.Mutex),
	}
	for _, opt := range opts {
		opt(o)
	}

	return &pinholeClient{
		vppConn: vppConn,
		mutex:   o.mutex,
	}
}

func (v *pinholeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	keys := []*IPPort{
		fromMechanism(conn.GetMechanism(), metadata.IsClient(v)),
		fromContext(ctx, metadata.IsClient(v)),
	}
	for _, key := range keys {
		if key == nil {
			continue
		}
		if _, ok := v.ipPortMap.LoadOrStore(*key, struct{}{}); !ok {
			v.mutex.Lock()
			if err := create(ctx, v.vppConn, key.IP(), key.Port(), fmt.Sprintf("%s port %d", aclTag, key.port)); err != nil {
				closeCtx, cancelClose := postponeCtxFunc()
				defer cancelClose()

				if _, closeErr := v.Close(closeCtx, conn, opts...); closeErr != nil {
					err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
				}

				v.mutex.Unlock()
				return nil, err
			}
			v.mutex.Unlock()
		}
	}

	return conn, nil
}

func (v *pinholeClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
