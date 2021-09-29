// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package initvpp

import (
	"context"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/acl"
	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

var ipV4zeroPrefix ip_types.Prefix = types.ToVppPrefix(&net.IPNet{
	IP:   net.IPv4zero,
	Mask: net.CIDRMask(0, 32),
})

var ipv6zeroPrefix ip_types.Prefix = types.ToVppPrefix(&net.IPNet{
	IP:   net.IPv6zero,
	Mask: net.CIDRMask(0, 128),
})

// DenyAllACLToInterface - deny ingress and egress ACL on the given interface
func DenyAllACLToInterface(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) error {
	now := time.Now()
	ingressRsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, ingressACLAddDelete())
	if err != nil {
		return errors.Wrapf(err, "unable to add denyall ACL")
	}

	log.FromContext(ctx).
		WithField("aclIndex", ingressRsp.ACLIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLAddReplace").Debug("completed")

	now = time.Now()
	egressRsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, egressACLAddDelete())
	if err != nil {
		return errors.Wrapf(err, "unable to add denyall ACL")
	}

	log.FromContext(ctx).
		WithField("aclIndex", egressRsp.ACLIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLAddReplace").Debug("completed")

	now = time.Now()
	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
		Count:     2,
		NInput:    1,
		Acls: []uint32{
			ingressRsp.ACLIndex,
			egressRsp.ACLIndex,
		},
	}
	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		return errors.Wrapf(err, "unable to add aclList ACL")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", interfaceACLList.SwIfIndex).
		WithField("acls", interfaceACLList.Acls).
		WithField("NInput", interfaceACLList.NInput).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLInterfaceSetACLList").Debug("completed")
	return nil
}

func ingressACLAddDelete() *acl.ACLAddReplace {
	return &acl.ACLAddReplace{
		ACLIndex: ^uint32(0),
		Tag:      "nsm-vppinit-denyall-ingress",
		R: []acl_types.ACLRule{
			{
				// Allow ingress ICMPv6 Router Advertisement Message
				IsPermit:               acl_types.ACL_ACTION_API_PERMIT,
				SrcPrefix:              ipv6zeroPrefix,
				Proto:                  ip_types.IP_API_PROTO_ICMP6,
				DstportOrIcmpcodeFirst: 134,
				DstportOrIcmpcodeLast:  134,
			},
			{
				// Allow ingress ICMPv6 Neighbor Advertisement Message
				IsPermit:               acl_types.ACL_ACTION_API_PERMIT,
				SrcPrefix:              ipv6zeroPrefix,
				Proto:                  ip_types.IP_API_PROTO_ICMP6,
				DstportOrIcmpcodeFirst: 136,
				DstportOrIcmpcodeLast:  136,
			},
			{
				IsPermit:  acl_types.ACL_ACTION_API_DENY,
				SrcPrefix: ipV4zeroPrefix,
			},
			{
				IsPermit:  acl_types.ACL_ACTION_API_DENY,
				SrcPrefix: ipv6zeroPrefix,
			},
		},
	}
}

func egressACLAddDelete() *acl.ACLAddReplace {
	return &acl.ACLAddReplace{
		ACLIndex: ^uint32(0),
		Tag:      "nsm-vppinit-denyall-egress",
		R: []acl_types.ACLRule{
			{
				// Allow egress ICMPv6 Router Solicitation Message
				IsPermit:               acl_types.ACL_ACTION_API_PERMIT,
				Proto:                  ip_types.IP_API_PROTO_ICMP6,
				DstPrefix:              ipv6zeroPrefix,
				SrcportOrIcmptypeFirst: 133,
				SrcportOrIcmptypeLast:  133,
			},
			{
				// Allow egress ICMPv6 Neighbor Solicitation Message
				IsPermit:               acl_types.ACL_ACTION_API_PERMIT,
				Proto:                  ip_types.IP_API_PROTO_ICMP6,
				SrcPrefix:              ipv6zeroPrefix,
				SrcportOrIcmptypeFirst: 135,
				SrcportOrIcmptypeLast:  135,
			},
			{
				IsPermit:  acl_types.ACL_ACTION_API_DENY,
				DstPrefix: ipV4zeroPrefix,
			},
			{
				IsPermit:  acl_types.ACL_ACTION_API_DENY,
				DstPrefix: ipv6zeroPrefix,
			},
		},
	}
}
