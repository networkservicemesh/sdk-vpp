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
	"strconv"
	"strings"

	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	action        = "action"     // DENY, PERMIT, REFLECT
	dstPrefix     = "dst_prefix" // IPv4 or IPv6 CIDR
	srcPrefix     = "src_prefix" // IPv4 or IPv6 CIDR
	protocol      = "proto"
	icmpTypeFirst = "icmp_type_first" // 8-bit unsigned integer
	icmpTypeLast  = "icmp_type_last"  // 8-bit unsigned integer
	icmpCodeFirst = "icmp_code_first" // 8-bit unsigned integer
	icmpCodeLast  = "icmp_code_last"  // 8-bit unsigned integer
	srcPortLast   = "src_port_last"   // 16-bit unsigned integer
	srcPortFirst  = "src_port_first"  // 16-bit unsigned integer
	dstPortLast   = "dst_port_last"   // 16-bit unsigned integer
	dstPortFirst  = "dst_port_first"  // 16-bit unsigned integer
	tcpFlagsMask  = "tcp_flags_mask"  // 8-bit unsigned integer
	tcpFlagsValue = "tcp_flags_value" // 8-bit unsigned integer
)

func parseACLRulesMap(ctx context.Context, rules map[string]string) (aclRules []acl_types.ACLRule) {
	logger := log.FromContext(ctx).WithField("acl", "parser")
	for _, r := range rules {
		rule := &acl_types.ACLRule{}

		parsed := parseKVStringToMap(r, ",", "=")

		if err := addAction(parsed, rule); err != nil {
			logger.Errorf("Error parsing action: %v", err.Error())
			continue
		}

		if err := addProtocol(parsed, rule); err != nil {
			logger.WithField("acl", "parser").Errorf("Error parsing protocol: %v", err.Error())
			continue
		}

		addPrefixes(parsed, rule)

		if err := addSrcPortOrIcmpType(parsed, rule); err != nil {
			logger.WithField("acl", "parser").Errorf("Error parsing srcPortOrIcmpType: %v", err.Error())
			continue
		}

		if err := addDstPortOrIcmpCode(parsed, rule); err != nil {
			logger.WithField("acl", "parser").Errorf("Error parsing dstPortOrIcmpType: %v", err.Error())
			continue
		}

		addTCPFlags(parsed, rule)

		aclRules = append(aclRules, *rule)
	}

	return aclRules
}

var actionMap = map[string]acl_types.ACLAction{
	"permit":  acl_types.ACL_ACTION_API_PERMIT,
	"reflect": acl_types.ACL_ACTION_API_PERMIT_REFLECT,
	"deny":    acl_types.ACL_ACTION_API_DENY,
}

func addAction(parsed map[string]string, rule *acl_types.ACLRule) error {
	actionName, ok := parsed[action]
	if !ok {
		return errors.New("action is missing")
	}

	a, ok := actionMap[actionName]
	if !ok {
		return errors.New("invalid action")
	}

	rule.IsPermit = a
	return nil
}

func addPrefixes(parsed map[string]string, rule *acl_types.ACLRule) {
	dst, dstOk := parsed[dstPrefix]
	if !dstOk {
		rule.DstPrefix = ip_types.Prefix{}
	} else {
		dstpref, err := ip_types.ParsePrefix(dst)
		if err != nil {
			rule.DstPrefix = ip_types.Prefix{}
		} else {
			rule.DstPrefix = dstpref
		}
	}

	src, srcOk := parsed[srcPrefix]
	if !srcOk {
		rule.SrcPrefix = ip_types.Prefix{}
	} else {
		srcpref, err := ip_types.ParsePrefix(src)
		if err != nil {
			rule.SrcPrefix = ip_types.Prefix{}
		} else {
			rule.SrcPrefix = srcpref
		}
	}
}

var protocolMap = map[string]ip_types.IPProto{
	"hopopt": ip_types.IP_API_PROTO_HOPOPT,
	"icmp":   ip_types.IP_API_PROTO_ICMP,
	"igmp":   ip_types.IP_API_PROTO_IGMP,
	"tcp":    ip_types.IP_API_PROTO_TCP,
	"udp":    ip_types.IP_API_PROTO_UDP,
	"gre":    ip_types.IP_API_PROTO_GRE,
	"esp":    ip_types.IP_API_PROTO_ESP,
	"ah":     ip_types.IP_API_PROTO_AH,
	"icmp6":  ip_types.IP_API_PROTO_ICMP6,
	"eigrp":  ip_types.IP_API_PROTO_EIGRP,
	"ospf":   ip_types.IP_API_PROTO_OSPF,
	"sctp":   ip_types.IP_API_PROTO_SCTP,
}

func addProtocol(parsed map[string]string, rule *acl_types.ACLRule) error {
	proto, ok := parsed[protocol]
	if !ok {
		return errors.New("protocol is missing")
	}

	p, ok := protocolMap[proto]
	if !ok {
		return errors.New("invalid protocol")
	}

	rule.Proto = p
	return nil
}

func addSrcPortOrIcmpType(parsed map[string]string, rule *acl_types.ACLRule) error {
	if rule.Proto == ip_types.IP_API_PROTO_ICMP || rule.Proto == ip_types.IP_API_PROTO_ICMP6 {
		first, last, err := findNumberPairByKeys(parsed, icmpTypeFirst, icmpTypeLast)
		if err != nil {
			return err
		}

		rule.SrcportOrIcmptypeFirst = first
		rule.SrcportOrIcmptypeLast = last

		return nil
	}

	first, last, err := findNumberPairByKeys(parsed, srcPortFirst, srcPortLast)
	if err != nil {
		return err
	}

	rule.SrcportOrIcmptypeFirst = first
	rule.SrcportOrIcmptypeLast = last

	return nil
}

func addDstPortOrIcmpCode(parsed map[string]string, rule *acl_types.ACLRule) error {
	if rule.Proto == ip_types.IP_API_PROTO_ICMP || rule.Proto == ip_types.IP_API_PROTO_ICMP6 {
		first, last, err := findNumberPairByKeys(parsed, icmpCodeFirst, icmpCodeLast)
		if err != nil {
			return err
		}

		rule.DstportOrIcmpcodeFirst = first
		rule.DstportOrIcmpcodeLast = last

		return nil
	}

	first, last, err := findNumberPairByKeys(parsed, dstPortFirst, dstPortLast)
	if err != nil {
		return err
	}

	rule.DstportOrIcmpcodeFirst = first
	rule.DstportOrIcmpcodeLast = last

	return nil
}

func addTCPFlags(parsed map[string]string, rule *acl_types.ACLRule) {
	tcpFlagsMask, ok := parsed[tcpFlagsMask]
	if !ok {
		return
	}
	tcpFlagsVal, ok := parsed[tcpFlagsValue]
	if !ok {
		return
	}

	mask, err := strconv.Atoi(tcpFlagsMask)
	if err != nil {
		return
	}
	val, err := strconv.Atoi(tcpFlagsVal)
	if err != nil {
		return
	}

	rule.TCPFlagsMask = uint8(mask)
	rule.TCPFlagsValue = uint8(val)
}

// parseKVStringToMap parses the input string "ke1${kvsep}val1${sep}key2${kvsep}val2${sep}..." to map
func parseKVStringToMap(input, sep, kvsep string) map[string]string {
	result := map[string]string{}
	pairs := strings.Split(input, sep)
	for _, pair := range pairs {
		k, v := parseKV(pair, kvsep)
		result[k] = v
	}
	return result
}

func parseKV(kv, kvsep string) (key, val string) {
	keyValue := strings.Split(kv, kvsep)
	if len(keyValue) != 2 {
		keyValue = []string{"", ""}
	}
	return strings.Trim(keyValue[0], " "), strings.Trim(keyValue[1], " ")
}

func findNumberPairByKeys(parsed map[string]string, key1, key2 string) (fisrtval, lastval uint16, err error) {
	first, ok := parsed[key1]
	if !ok {
		return 0, 0, errors.New("icmp code first is missing")
	}

	last, ok := parsed[key2]
	if !ok {
		return 0, 0, errors.New("icmp code last is missing")
	}

	numFirst, err := strconv.Atoi(first)
	if err != nil {
		return 0, 0, err
	}
	numLast, err := strconv.Atoi(last)
	if err != nil {
		return 0, 0, err
	}

	return uint16(numFirst), uint16(numLast), nil
}
