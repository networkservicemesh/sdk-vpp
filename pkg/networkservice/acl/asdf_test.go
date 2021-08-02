package acl

import (
	"github.com/edwarnicke/govpp/binapi/acl_types"
	"github.com/edwarnicke/govpp/binapi/ip_types"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

type a struct {
	ACLRules map[string]acl_types.ACLRule
}

func TestAsdf(t *testing.T){
	file, err := os.Create("test2.yaml")
	require.NoError(t, err)
	require.NotNil(t, file)

	A := a {
		ACLRules: map[string]acl_types.ACLRule{
			"allow icmp" : getSpecificICMPRule2(),
		},
	}
	bytes, err := yaml.Marshal(A)
	require.NoError(t, err)
	require.NotNil(t, bytes)
	require.Greater(t, len(bytes), 0)

	_, err = file.Write(bytes)
	require.NoError(t, err)
}

func TestV(t *testing.T) {
	raw, err := ioutil.ReadFile("test.yaml")
	require.NoError(t, err)
	require.NotNil(t, raw)

	var rv map[string]acl_types.ACLRule
	e := yaml.Unmarshal(raw, &rv)
	require.NoError(t, e)
	require.Equal(t, map[string]acl_types.ACLRule{
			"allow icmp" : getSpecificICMPRule2(),
		}, rv)
}

func getSpecificICMPRule2() acl_types.ACLRule {
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
