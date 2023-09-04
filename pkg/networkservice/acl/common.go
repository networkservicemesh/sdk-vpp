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

package acl

import (
	"context"
	"fmt"
	"time"

	"github.com/networkservicemesh/govpp/binapi/acl"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	aclTag = "nsm-acl-from-config"
)

func create(ctx context.Context, vppConn api.Connection, tag string, isClient bool, aRules []acl_types.ACLRule) ([]uint32, error) {
	logger := log.FromContext(ctx).WithField("acl_server", "create")

	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		logger.Info("swIfIndex not found")
		return nil, errors.New("swIfIndex not found")
	}
	logger.Infof(fmt.Sprintf("swIfIndex=%v", swIfIndex))

	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
	}

	var err error
	interfaceACLList.Acls, err = addACLToACLList(ctx, vppConn, tag, false, aRules)
	if err != nil {
		logger.Info("error adding acl to acl list ingress")
		return nil, err
	}
	interfaceACLList.NInput = uint8(len(interfaceACLList.Acls))

	egressACLIndeces, err := addACLToACLList(ctx, vppConn, tag, true, aRules)
	if err != nil {
		logger.Info("error adding acl to acl list egress")
		return nil, err
	}
	interfaceACLList.Acls = append(interfaceACLList.Acls, egressACLIndeces...)
	interfaceACLList.Count = uint8(len(interfaceACLList.Acls))

	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		logger.Info("error setting acl list for interface")
		return nil, errors.Wrap(err, "vppapi ACLInterfaceSetACLList returned error")
	}
	return interfaceACLList.Acls, nil
}

func addACLToACLList(ctx context.Context, vppConn api.Connection, tag string, egress bool, aRules []acl_types.ACLRule) ([]uint32, error) {
	var ACLIndeces []uint32

	now := time.Now()
	rsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, aclAdd(tag, egress, aRules))
	if err != nil {
		return nil, errors.Wrap(err, "vppapi ACLAddReplace returned error")
	}
	log.FromContext(ctx).
		WithField("aclIndices", rsp.ACLIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLAddReplace").Debug("completed")
	ACLIndeces = append([]uint32{rsp.ACLIndex}, ACLIndeces...)

	return ACLIndeces, nil
}

func aclAdd(tag string, egress bool, aRules []acl_types.ACLRule) *acl.ACLAddReplace {
	aclAddReplace := &acl.ACLAddReplace{
		ACLIndex: ^uint32(0),
		Tag:      tag,
		Count:    uint32(len(aRules)),
		R:        aRules,
	}
	if egress {
		for i := range aclAddReplace.R {
			aclAddReplace.R[i].SrcPrefix, aclAddReplace.R[i].DstPrefix =
				aclAddReplace.R[i].DstPrefix, aclAddReplace.R[i].SrcPrefix

			aclAddReplace.R[i].SrcportOrIcmptypeFirst, aclAddReplace.R[i].DstportOrIcmpcodeFirst =
				aclAddReplace.R[i].DstportOrIcmpcodeFirst, aclAddReplace.R[i].SrcportOrIcmptypeFirst

			aclAddReplace.R[i].SrcportOrIcmptypeLast, aclAddReplace.R[i].DstportOrIcmpcodeLast =
				aclAddReplace.R[i].DstportOrIcmpcodeLast, aclAddReplace.R[i].SrcportOrIcmptypeLast
		}
	}
	return aclAddReplace
}
