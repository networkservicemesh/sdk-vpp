// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package acl provides chain elements for setting acl rules
package acl

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/govpp/binapi/acl"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type aclServer struct {
	vppConn    api.Connection
	aclRules   []acl_types.ACLRule
	aclIndices aclIndicesMap
}

// NewServer creates a NetworkServiceServer chain element to set the ACL on a vpp interface
func NewServer(vppConn api.Connection, aclrules []acl_types.ACLRule) networkservice.NetworkServiceServer {
	return &aclServer{
		vppConn:  vppConn,
		aclRules: aclrules,
	}
}

func (a *aclServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	_, loaded := a.aclIndices.Load(conn.GetId())
	if !loaded && len(a.aclRules) > 0 {
		var indices []uint32
		if indices, err = create(ctx, a.vppConn, aclTag, metadata.IsClient(a), a.aclRules); err != nil {
			closeCtx, cancelClose := postponeCtxFunc()
			defer cancelClose()

			if _, closeErr := a.Close(closeCtx, conn); closeErr != nil {
				err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
			}

			return nil, err
		}

		a.aclIndices.Store(conn.GetId(), indices)
	}

	return conn, nil
}

func (a *aclServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	indices, _ := a.aclIndices.LoadAndDelete(conn.GetId())
	for ind := range indices {
		_, err := acl.NewServiceClient(a.vppConn).ACLDel(ctx, &acl.ACLDel{ACLIndex: uint32(ind)})
		if err != nil {
			log.FromContext(ctx).Infof("ACL_SERVER: error deleting acls")
		}
	}

	return next.Server(ctx).Close(ctx, conn)
}
