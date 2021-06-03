// Copyright (c) 2021 Doc.ai and/or its affiliates.
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
	"sync"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/acl"
	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type aclServer struct {
	vppConn    api.Connection
	aclRules   []acl_types.ACLRule
	aclIndices []uint32
	mut        sync.Mutex
}

// NewServer creates a NetworkServiceServer chain element to set the ACL on a vpp interface
func NewServer(vppConn api.Connection, aclrules []acl_types.ACLRule) networkservice.NetworkServiceServer {
	return &aclServer{
		vppConn:    vppConn,
		aclRules:   aclrules,
		aclIndices: make([]uint32, 0),
	}
}

func (a *aclServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(a.aclRules) > 0 {
		var indices []uint32
		if indices, err = create(ctx, a.vppConn, aclTag, metadata.IsClient(a), a.aclRules); err != nil {
			_, _ = a.Close(ctx, conn)
			return nil, errors.WithStack(err)
		}

		a.mut.Lock()
		a.aclIndices = indices
		a.mut.Unlock()
	}

	return conn, nil
}

func (a *aclServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	a.mut.Lock()
	for ind := range a.aclIndices {
		_, err := acl.NewServiceClient(a.vppConn).ACLDel(ctx, &acl.ACLDel{ACLIndex: uint32(ind)})
		if err != nil {
			log.FromContext(ctx).Infof("ACL_SERVER: error deleting acls")
		}
	}
	a.mut.Unlock()

	return next.Server(ctx).Close(ctx, conn)
}
