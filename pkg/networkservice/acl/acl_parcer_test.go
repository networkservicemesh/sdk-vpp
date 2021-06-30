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

package acl

import (
	"context"
	"testing"

	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/stretchr/testify/require"
)

type test struct {
	name       string
	inputRules map[string]string
	want       []acl_types.ACLRule
}

func TestAclParser(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tests := getTests()

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rules := parseACLRulesMap(ctx, tc.inputRules)

			require.Equal(t, len(tc.want), len(rules))

			for i := range rules {
				require.Equal(t, tc.want[i], rules[i])
			}
		})
	}
}

func getTests() []test {
	return []test{
		{
			name:       "empty",
			inputRules: map[string]string{},
			want:       []acl_types.ACLRule{},
		},
		{
			name: "correct general rule",
			inputRules: map[string]string{
				"Allow ICMP": "action=permit,proto=icmp,src_prefix=0.0.0.0/0,dst_prefix=0.0.0.0/0,icmp_type_first=0,icmp_type_last=65535,icmp_code_first=0,icmp_code_last=65535,tcp_flags_mask=0,tcp_flags_value=0",
			},
			want: []acl_types.ACLRule{getGeneralICMPrule()},
		},
		{
			name: "correct specific prefixes",
			inputRules: map[string]string{
				"Allow specific ICMP": "action=permit,proto=icmp,src_prefix=172.16.1.100/31,dst_prefix=172.16.1.101/31,icmp_type_first=0,icmp_type_last=65535,icmp_code_first=0,icmp_code_last=65535,tcp_flags_mask=0,tcp_flags_value=0",
			},
			want: []acl_types.ACLRule{getSpecificICMPRule()},
		},
		{
			name: "invalid/missing keys or values",
			inputRules: map[string]string{
				"Allow ICMP": "action=permit, proto=icmp",
				"Allow tcp":  "action=,proto=tcp,src_prefix=0.0.0.0/0,dst_prefix=0.0.0.0/0,src_port_first=0,src_port_last=65535,dst_port_first=8080,dst_port_last=8080,tcp_flags_mask=0,tcp_flags_value=0",
			},
			want: []acl_types.ACLRule{},
		},
		{
			name: "invalid values",
			inputRules: map[string]string{
				"Allow tcp": "action=somethingweird,proto=tcp,src_prefix=0.0.0.0/0,dst_prefix=0.0.0.0/0,src_port_first=0,src_port_last=65535,dst_port_first=8080,dst_port_last=8080,tcp_flags_mask=0,tcp_flags_value=0",
			},
			want: []acl_types.ACLRule{},
		},
	}
}

func getGeneralICMPrule() acl_types.ACLRule {
	return acl_types.ACLRule{
		IsPermit: acl_types.ACL_ACTION_API_PERMIT,
		SrcPrefix: ip_types.Prefix{
			Address: ip_types.Address{
				Af: 0,
				Un: ip_types.AddressUnion{},
			},
			Len: 0,
		},
		DstPrefix: ip_types.Prefix{
			Address: ip_types.Address{
				Af: 0,
				Un: ip_types.AddressUnion{},
			},
			Len: 0,
		},
		Proto:                  ip_types.IP_API_PROTO_ICMP,
		SrcportOrIcmptypeFirst: 0,
		SrcportOrIcmptypeLast:  65535,
		DstportOrIcmpcodeFirst: 0,
		DstportOrIcmpcodeLast:  65535,
		TCPFlagsMask:           0,
		TCPFlagsValue:          0,
	}
}

func getSpecificICMPRule() acl_types.ACLRule {
	return acl_types.ACLRule{
		IsPermit: acl_types.ACL_ACTION_API_PERMIT,
		SrcPrefix: ip_types.Prefix{
			Address: ip_types.Address{
				Af: 0,
				Un: ip_types.AddressUnion{
					XXX_UnionData: [16]byte{
						172, 16, 1, 100,
					},
				},
			},
			Len: 31,
		},
		DstPrefix: ip_types.Prefix{
			Address: ip_types.Address{
				Af: 0,
				Un: ip_types.AddressUnion{
					XXX_UnionData: [16]byte{
						172, 16, 1, 101,
					},
				},
			},
			Len: 31,
		},
		Proto:                  ip_types.IP_API_PROTO_ICMP,
		SrcportOrIcmptypeFirst: 0,
		SrcportOrIcmptypeLast:  65535,
		DstportOrIcmpcodeFirst: 0,
		DstportOrIcmpcodeLast:  65535,
		TCPFlagsMask:           0,
		TCPFlagsValue:          0,
	}
}
