// Copyright (c) 2020 Cisco and/or its affiliates.
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

package vxlanacl

import (
	"context"
	"net"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/govpp/binapi/acl"
	"github.com/edwarnicke/govpp/binapi/acl_types"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/ip"
	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

const (
	aclTag = "nsm-mechanism-vxlan"
)

func create(ctx context.Context, vppConn api.Connection, tunnelIP net.IP, tag string) error {
	swIfIndex, err := vxlanSwIfIndex(ctx, vppConn, tunnelIP)
	if err != nil {
		return errors.WithStack(err)
	}

	ingressACLs, egressACLs, err := interfacesACLDetails(ctx, vppConn, swIfIndex)
	if err != nil {
		return errors.WithStack(err)
	}

	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
	}

	interfaceACLList.Acls, err = addToVxlanACLToACLListIfNeeded(ctx, vppConn, tunnelIP, tag, false, ingressACLs)
	if err != nil {
		return errors.WithStack(err)
	}
	interfaceACLList.NInput = uint8(len(interfaceACLList.Acls))

	egressACLIndeces, err := addToVxlanACLToACLListIfNeeded(ctx, vppConn, tunnelIP, tag, true, egressACLs)
	if err != nil {
		return errors.WithStack(err)
	}
	interfaceACLList.Acls = append(interfaceACLList.Acls, egressACLIndeces...)
	interfaceACLList.Count = uint8(len(interfaceACLList.Acls))

	if interfaceACLList.Count == uint8(len(ingressACLs)+len(egressACLs)) {
		return nil
	}
	now := time.Now()
	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		return errors.WithStack(err)
	}
	logger.Log(ctx).
		WithField("swIfIndex", interfaceACLList.SwIfIndex).
		WithField("acls", interfaceACLList.Acls).
		WithField("NInput", interfaceACLList.NInput).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLInterfaceSetACLList").Debug("completed")
	return nil
}

func addToVxlanACLToACLListIfNeeded(ctx context.Context, vppConn api.Connection, tunnelIP net.IP, tag string, egress bool, aclDetails []*acl.ACLDetails) ([]uint32, error) {
	var foundACL *acl.ACLDetails
	var ACLIndeces []uint32
	for _, aclDetail := range aclDetails {
		ACLIndeces = append(ACLIndeces, aclDetail.ACLIndex)
		if aclDetail.Tag == tag {
			foundACL = aclDetail
		}
	}

	if foundACL == nil && len(aclDetails) > 0 {
		now := time.Now()
		rsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, vxlanACL(tunnelIP, tag, egress))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		logger.Log(ctx).
			WithField("aclIndex", rsp.ACLIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "ACLAddReplace").Debug("completed")
		ACLIndeces = append([]uint32{rsp.ACLIndex}, ACLIndeces...)
	}
	return ACLIndeces, nil
}

func interfacesACLDetails(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) (ingressACLs, egressACLs []*acl.ACLDetails, err error) {
	ingressIndeces, egressIndeces, err := interfaceACLIndeces(ctx, vppConn, swIfIndex)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	ingressACLs, err = aclDetails(ctx, vppConn, ingressIndeces)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	egressACLs, err = aclDetails(ctx, vppConn, egressIndeces)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return ingressACLs, egressACLs, nil
}

func aclDetails(ctx context.Context, vppConn api.Connection, aclIndeces []uint32) ([]*acl.ACLDetails, error) {
	var rv []*acl.ACLDetails
	for _, aclIndex := range aclIndeces {
		now := time.Now()
		aclDumpClient, err := acl.NewServiceClient(vppConn).ACLDump(ctx, &acl.ACLDump{
			ACLIndex: aclIndex,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		aclDetails, err := aclDumpClient.Recv()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		logger.Log(ctx).
			WithField("aclIndex", aclIndex).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "ACLDump").Debugf("completed")
		rv = append(rv, aclDetails)
	}
	return rv, nil
}

func interfaceACLIndeces(ctx context.Context, vppConn api.Connection, swIfIndex interface_types.InterfaceIndex) (ingressACLIndeces, egressACLIndeces []uint32, err error) {
	now := time.Now()
	aclInterfaceListDumpClient, err := acl.NewServiceClient(vppConn).ACLInterfaceListDump(ctx, &acl.ACLInterfaceListDump{
		SwIfIndex: swIfIndex,
	})
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	defer func() { _ = aclInterfaceListDumpClient.Close() }()
	aclInterfaceListDetails, err := aclInterfaceListDumpClient.Recv()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	logger.Log(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLInterfaceListDump").Debug("initiated")
	ingressACLIndeces = aclInterfaceListDetails.Acls[:aclInterfaceListDetails.NInput]
	egressACLIndeces = aclInterfaceListDetails.Acls[aclInterfaceListDetails.NInput:]
	return ingressACLIndeces, egressACLIndeces, nil
}

func vxlanSwIfIndex(ctx context.Context, vppConn api.Connection, tunnelIP net.IP) (interface_types.InterfaceIndex, error) {
	now := time.Now()
	swIfDumpClient, swIfDumpErr := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if swIfDumpErr != nil {
		return 0, errors.WithStack(swIfDumpErr)
	}
	defer func() { _ = swIfDumpClient.Close() }()
	swIfDetails, swIfDumpErr := swIfDumpClient.Recv()
	for swIfDumpErr == nil {
		ipDumpClient, err := ip.NewServiceClient(vppConn).IPAddressDump(ctx, &ip.IPAddressDump{
			SwIfIndex: swIfDetails.SwIfIndex,
			IsIPv6:    tunnelIP.To4() == nil,
		})
		if err != nil {
			return 0, errors.WithStack(err)
		}
		defer func() { _ = ipDumpClient.Close() }()
		ipDetails, ipDumpErr := ipDumpClient.Recv()
		for ipDumpErr == nil {
			if types.FromVppAddressWithPrefix(ipDetails.Prefix).IP.Equal(tunnelIP) {
				logger.Log(ctx).
					WithField("swIfIndex", swIfDetails.SwIfIndex).
					WithField("duration", time.Since(now)).
					WithField("vppapi", "SwInterfaceDump").Debugf("found interface with ip %s", tunnelIP)
				return swIfDetails.SwIfIndex, nil
			}
			ipDetails, ipDumpErr = ipDumpClient.Recv()
		}
		swIfDetails, swIfDumpErr = swIfDumpClient.Recv()
	}
	logger.Log(ctx).
		WithField("swIfIndex", swIfDetails.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceDump").Debugf("did not find interface with ip %s", tunnelIP)
	return 0, errors.Errorf("unable to find tunnelIP (%s) on any vpp interface", tunnelIP)
}

func vxlanACL(tunnelIP net.IP, tag string, egress bool) *acl.ACLAddReplace {
	defaultNet := &net.IPNet{
		IP:   net.IPv4zero,
		Mask: net.CIDRMask(0, 32),
	}
	tunnelNet := &net.IPNet{
		IP:   tunnelIP,
		Mask: net.CIDRMask(32, 32),
	}
	if tunnelIP.To4() == nil {
		defaultNet = &net.IPNet{
			IP:   net.IPv6zero,
			Mask: net.CIDRMask(0, 128),
		}
		tunnelNet = &net.IPNet{
			IP:   tunnelIP,
			Mask: net.CIDRMask(128, 128),
		}
	}
	aclAddDelete := &acl.ACLAddReplace{
		ACLIndex: ^uint32(0),
		Tag:      tag,
		Count:    1,
		R: []acl_types.ACLRule{
			{
				IsPermit:               acl_types.ACL_ACTION_API_PERMIT,
				Proto:                  ip_types.IP_API_PROTO_UDP,
				SrcPrefix:              types.ToVppPrefix(defaultNet),
				DstPrefix:              types.ToVppPrefix(tunnelNet),
				SrcportOrIcmptypeFirst: 0,
				SrcportOrIcmptypeLast:  65535,
				DstportOrIcmpcodeFirst: 4789,
				DstportOrIcmpcodeLast:  4789,
			},
		},
	}
	if egress {
		aclAddDelete.R[0].SrcPrefix = types.ToVppPrefix(tunnelNet)
		aclAddDelete.R[0].DstPrefix = types.ToVppPrefix(defaultNet)
	}
	return aclAddDelete
}
