// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package pinhole

import (
	"context"
	"net"
	"time"

	"github.com/networkservicemesh/govpp/binapi/acl"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip"
	"github.com/networkservicemesh/govpp/binapi/ip_types"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"
)

const (
	aclTag = "nsm-pinhole"
)

func create(ctx context.Context, vppConn api.Connection, tunnelIP net.IP, port uint16, tag string) error {
	if tunnelIP == nil || port == 0 {
		return nil
	}
	swIfIndex, err := tunnelIPSwIfIndex(ctx, vppConn, tunnelIP)
	if err != nil {
		return err
	}

	ingressACLs, egressACLs, err := interfacesACLDetails(ctx, vppConn, swIfIndex)
	if err != nil {
		return err
	}

	interfaceACLList := &acl.ACLInterfaceSetACLList{
		SwIfIndex: swIfIndex,
	}

	interfaceACLList.Acls, err = addToACLToACLListIfNeeded(ctx, vppConn, tunnelIP, port, tag, false, ingressACLs)
	if err != nil {
		return err
	}
	interfaceACLList.NInput = uint8(len(interfaceACLList.Acls))

	egressACLIndeces, err := addToACLToACLListIfNeeded(ctx, vppConn, tunnelIP, port, tag, true, egressACLs)
	if err != nil {
		return err
	}
	interfaceACLList.Acls = append(interfaceACLList.Acls, egressACLIndeces...)
	interfaceACLList.Count = uint8(len(interfaceACLList.Acls))

	if interfaceACLList.Count == uint8(len(ingressACLs)+len(egressACLs)) {
		return nil
	}
	now := time.Now()
	_, err = acl.NewServiceClient(vppConn).ACLInterfaceSetACLList(ctx, interfaceACLList)
	if err != nil {
		return errors.Wrap(err, "vppapi ACLInterfaceSetACLList returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", interfaceACLList.SwIfIndex).
		WithField("acls", interfaceACLList.Acls).
		WithField("NInput", interfaceACLList.NInput).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLInterfaceSetACLList").Debug("completed")
	return nil
}

func addToACLToACLListIfNeeded(ctx context.Context, vppConn api.Connection, tunnelIP net.IP, port uint16, tag string, egress bool, aclDetails []*acl.ACLDetails) ([]uint32, error) {
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
		rsp, err := acl.NewServiceClient(vppConn).ACLAddReplace(ctx, createACLAddReplace(tunnelIP, port, tag, egress))
		if err != nil {
			return nil, errors.Wrap(err, "vppapi ACLAddReplace returned error")
		}
		log.FromContext(ctx).
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
		return nil, nil, err
	}
	ingressACLs, err = aclDetails(ctx, vppConn, ingressIndeces)
	if err != nil {
		return nil, nil, err
	}
	egressACLs, err = aclDetails(ctx, vppConn, egressIndeces)
	if err != nil {
		return nil, nil, err
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
			return nil, errors.Wrap(err, "vppapi ACLDump returned error")
		}
		defer func() { _ = aclDumpClient.Close() }()
		aclDetails, err := aclDumpClient.Recv()
		if err != nil {
			return nil, errors.Wrapf(err, "error retrieving aclDetails")
		}
		log.FromContext(ctx).
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
		return nil, nil, errors.Wrap(err, "vppapi ACLInterfaceListDump returned error")
	}
	defer func() { _ = aclInterfaceListDumpClient.Close() }()
	aclInterfaceListDetails, err := aclInterfaceListDumpClient.Recv()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error retrieving aclInterfaceListDetails for swIfIndex %d", swIfIndex)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "ACLInterfaceListDump").Debug("initiated")
	ingressACLIndeces = aclInterfaceListDetails.Acls[:aclInterfaceListDetails.NInput]
	egressACLIndeces = aclInterfaceListDetails.Acls[aclInterfaceListDetails.NInput:]
	return ingressACLIndeces, egressACLIndeces, nil
}

func tunnelIPSwIfIndex(ctx context.Context, vppConn api.Connection, tunnelIP net.IP) (interface_types.InterfaceIndex, error) {
	now := time.Now()
	swIfDumpClient, swIfDumpErr := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if swIfDumpErr != nil {
		return 0, errors.Wrap(swIfDumpErr, "vppapi SwInterfaceDump returned error")
	}
	defer func() { _ = swIfDumpClient.Close() }()
	swIfDetails, swIfDumpErr := swIfDumpClient.Recv()
	for swIfDumpErr == nil {
		ipDumpClient, err := ip.NewServiceClient(vppConn).IPAddressDump(ctx, &ip.IPAddressDump{
			SwIfIndex: swIfDetails.SwIfIndex,
			IsIPv6:    tunnelIP.To4() == nil,
		})
		if err != nil {
			return 0, errors.Wrap(err, "vppapi IPAddressDump returned error")
		}
		defer func() { _ = ipDumpClient.Close() }()
		ipDetails, ipDumpErr := ipDumpClient.Recv()
		for ipDumpErr == nil {
			if types.FromVppAddressWithPrefix(ipDetails.Prefix).IP.Equal(tunnelIP) {
				log.FromContext(ctx).
					WithField("swIfIndex", swIfDetails.SwIfIndex).
					WithField("duration", time.Since(now)).
					WithField("vppapi", "SwInterfaceDump").Debugf("found interface with ip %s", tunnelIP)
				return swIfDetails.SwIfIndex, nil
			}
			ipDetails, ipDumpErr = ipDumpClient.Recv()
		}
		swIfDetails, swIfDumpErr = swIfDumpClient.Recv()
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfDetails.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceDump").Debugf("did not find interface with ip %s", tunnelIP)
	return 0, errors.Errorf("unable to find tunnelIP (%s) on any vpp interface", tunnelIP)
}

func createACLAddReplace(tunnelIP net.IP, port uint16, tag string, egress bool) *acl.ACLAddReplace {
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
	aclAddReplace := &acl.ACLAddReplace{
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
				DstportOrIcmpcodeFirst: port,
				DstportOrIcmpcodeLast:  port,
			},
		},
	}
	if egress {
		aclAddReplace.R[0].SrcPrefix = types.ToVppPrefix(tunnelNet)
		aclAddReplace.R[0].DstPrefix = types.ToVppPrefix(defaultNet)
	}
	return aclAddReplace
}
