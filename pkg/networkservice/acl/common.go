// Copyright (c) 2021 Cisco and/or its affiliates.
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
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/acl"
	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	aclTag = "nsm-acl-from-config"
)

func create(ctx context.Context, vppConn api.Connection, tag string, swIfIndex interface_types.InterfaceIndex, aRules []acl_types.ACLRule) ([]uint32, error) {
	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
	}

	var err error
	interfaceACLList.Acls, err = addACLToACLList(ctx, vppConn, tag, false, aRules)
	if err != nil {
		log.FromContext(ctx).Info("ACL_SERVER: error adding acl to acl list ingress")
		return nil, errors.WithStack(err)
	}
	interfaceACLList.NInput = uint8(len(interfaceACLList.Acls))

	egressACLIndeces, err := addACLToACLList(ctx, vppConn, tag, true, aRules)
	if err != nil {
		log.FromContext(ctx).Info("ACL_SERVER: error adding acl to acl list egress")
		return nil, errors.WithStack(err)
	}
	interfaceACLList.Acls = append(interfaceACLList.Acls, egressACLIndeces...)
	interfaceACLList.Count = uint8(len(interfaceACLList.Acls))

	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		log.FromContext(ctx).Info("ACL_SERVER: error setting acl list for interface")
		return nil, errors.WithStack(err)
	}
	return interfaceACLList.Acls, nil
}

func addACLToACLList(ctx context.Context, vppConn api.Connection, tag string, egress bool, aRules []acl_types.ACLRule) ([]uint32, error) {
	var ACLIndeces []uint32

	now := time.Now()
	rsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, aclAddDel(tag, egress, aRules))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("aclIndices", rsp.ACLIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLAddReplace").Debug("completed")
	ACLIndeces = append([]uint32{rsp.ACLIndex}, ACLIndeces...)

	return ACLIndeces, nil
}

func aclAddDel(tag string, egress bool, aRules []acl_types.ACLRule) *acl.ACLAddReplace {
	aclAddDelete := &acl.ACLAddReplace{
		ACLIndex: ^uint32(0),
		Tag:      tag,
		Count:    uint32(len(aRules)),
		R:        aRules,
	}
	if egress {
		for _, a := range aRules {
			a.SrcPrefix, a.DstPrefix = a.DstPrefix, a.SrcPrefix
		}
	}
	return aclAddDelete
}
