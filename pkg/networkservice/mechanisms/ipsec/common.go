// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package ipsec

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path"
	"time"

	"github.com/networkservicemesh/govpp/binapi/ikev2_types"
	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip"
	"github.com/networkservicemesh/govpp/binapi/ip_types"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/ipsec"
	"github.com/networkservicemesh/govpp/binapi/ikev2"
	ipsecapi "github.com/networkservicemesh/govpp/binapi/ipsec"
)

// create - creates IPSEC with IKEv2
func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, privateKey *rsa.PrivateKey, isClient bool) error {
	if mechanism := ipsec.ToMechanism(conn.GetMechanism()); mechanism != nil {
		_, ok := ifindex.Load(ctx, isClient)
		if ok {
			return nil
		}
		profileName := fmt.Sprintf("%s-%s", isClientPrefix(isClient), conn.Id)

		// *** CREATE IP TUNNEL *** //
		swIfIndex, err := createIPSecTunnel(ctx, vppConn)
		if err != nil {
			return err
		}

		// *** CREATE PROFILE *** //
		err = addDelProfile(ctx, vppConn, profileName, true)
		if err != nil {
			return err
		}

		// *** SET UDP ENCAPSULATION *** //
		err = setUDPEncap(ctx, vppConn, profileName)
		if err != nil {
			return err
		}

		// *** SET KEYS *** //
		err = setKeys(ctx, vppConn, profileName, mechanism, privateKey, isClient)
		if err != nil {
			return err
		}

		// *** SET FQDN *** //
		err = setFQDN(ctx, vppConn, mechanism, profileName, isClient)
		if err != nil {
			return err
		}

		// *** SET TRAFFIC-SELECTOR *** //
		err = setTrafficSelector(ctx, vppConn, profileName, conn, isClient)
		if err != nil {
			return err
		}

		// *** PROTECT THE TUNNEL *** //
		err = protectTunnel(ctx, vppConn, profileName, swIfIndex)
		if err != nil {
			return err
		}

		// *** INITIATOR STEPS *** //
		if isClient {
			err = initiate(ctx, vppConn, mechanism, profileName)
			if err != nil {
				return err
			}
		}

		ifindex.Store(ctx, isClient, swIfIndex)
	}
	return nil
}

func initiate(ctx context.Context, vppConn api.Connection, mechanism *ipsec.Mechanism, profileName string) error {
	hostSwIfIndex, err := getSwIfIndexByIP(ctx, vppConn, mechanism.SrcIP())
	if err != nil {
		return err
	}

	// *** SET RESPONDER *** //
	err = setResponder(ctx, vppConn, profileName, hostSwIfIndex, mechanism.DstIP())
	if err != nil {
		return err
	}

	// *** SET TRANSFORMS *** //
	err = setTransforms(ctx, vppConn, profileName)
	if err != nil {
		return err
	}

	// *** SET SA LIFETIME *** //
	err = setSaLifetime(ctx, vppConn, profileName)
	if err != nil {
		return err
	}

	// *** START INITIATION *** //
	err = saInit(ctx, vppConn, profileName)
	if err != nil {
		return err
	}

	return nil
}

func getSwIfIndexByIP(ctx context.Context, vppConn api.Connection, interfaceIP net.IP) (interface_types.InterfaceIndex, error) {
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if err != nil {
		return 0, errors.Wrapf(err, "error attempting to get interface dump client for IP %q", interfaceIP)
	}
	defer func() { _ = client.Close() }()

	for {
		details, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrapf(err, "error attempting to get interface details for IP %q", interfaceIP)
		}

		ipAddressClient, err := ip.NewServiceClient(vppConn).IPAddressDump(ctx, &ip.IPAddressDump{
			SwIfIndex: details.SwIfIndex,
			IsIPv6:    interfaceIP.To4() == nil,
		})
		if err != nil {
			return 0, errors.Wrapf(err, "error attempting to get ip address dump client for vpp interface %q ", details.InterfaceName)
		}
		defer func() { _ = ipAddressClient.Close() }()

		for {
			ipAddressDetails, err := ipAddressClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, errors.Wrapf(err, "error attempting to get ip address for %q (swIfIndex: %q)", details.InterfaceName, details.SwIfIndex)
			}
			if types.FromVppAddressWithPrefix(ipAddressDetails.Prefix).IP.Equal(interfaceIP) {
				return details.SwIfIndex, nil
			}
		}
	}
	return interface_types.InterfaceIndex(^uint32(0)), errors.Errorf("unable to find interface in vpp with IP: %q", interfaceIP)
}

func createIPSecTunnel(ctx context.Context, vppConn api.Connection) (interface_types.InterfaceIndex, error) {
	now := time.Now()

	reply, err := ipsecapi.NewServiceClient(vppConn).IpsecItfCreate(ctx, &ipsecapi.IpsecItfCreate{
		Itf: ipsecapi.IpsecItf{UserInstance: ^uint32(0)}})

	if err != nil {
		return interface_types.InterfaceIndex(^uint32(0)), errors.Wrap(err, "vppapi IpsecItfCreate returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", reply.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IpsecItfCreate").Debug("completed")
	return reply.SwIfIndex, nil
}

func delIPSecTunnel(ctx context.Context, vppConn api.Connection, isClient bool) error {
	now := time.Now()
	swIfIndex, ok := ifindex.LoadAndDelete(ctx, isClient)
	if !ok {
		return nil
	}

	_, err := ipsecapi.NewServiceClient(vppConn).IpsecItfDelete(ctx, &ipsecapi.IpsecItfDelete{SwIfIndex: swIfIndex})
	if err != nil {
		return errors.Wrap(err, "vppapi IpsecItfDelete returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", swIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "IpsecItfDelete").Debug("completed")
	return nil
}

func addDelProfile(ctx context.Context, vppConn api.Connection, profileName string, isAdd bool) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2ProfileAddDel(ctx, &ikev2.Ikev2ProfileAddDel{
		Name:  profileName,
		IsAdd: isAdd,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2ProfileAddDel returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("IsAdd", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2ProfileAddDel").Debug("completed")
	return nil
}

func setUDPEncap(ctx context.Context, vppConn api.Connection, profileName string) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2ProfileSetUDPEncap(ctx, &ikev2.Ikev2ProfileSetUDPEncap{
		Name: profileName,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2ProfileSetUDPEncap returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2ProfileSetUDPEncap").Debug("completed")
	return nil
}

func setKeys(ctx context.Context, vppConn api.Connection, profileName string, mechanism *ipsec.Mechanism, privateKey *rsa.PrivateKey, isClient bool) error {
	publicKeyBase64 := mechanism.SrcPublicKey()
	if isClient {
		publicKeyBase64 = mechanism.DstPublicKey()
	}
	publicKeyFileName, err := dumpCertBase64ToFile(publicKeyBase64, profileName, isClient)
	if err != nil {
		return err
	}
	log.FromContext(ctx).WithField("operation", "dumpCertBase64ToFile").Debug("completed")

	privateKeyFileName, err := dumpPrivateKeyToFile(privateKey, profileName, isClient)
	if err != nil {
		return err
	}
	log.FromContext(ctx).WithField("operation", "dumpPrivateKeyToFile").Debug("completed")

	now := time.Now()
	_, err = ikev2.NewServiceClient(vppConn).Ikev2ProfileSetAuth(ctx, &ikev2.Ikev2ProfileSetAuth{
		Name:       profileName,
		AuthMethod: 1, // rsa-sig
		DataLen:    uint32(len(publicKeyFileName)),
		Data:       []byte(publicKeyFileName),
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2ProfileSetAuth returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2ProfileSetAuth").Debug("completed")

	now = time.Now()
	_, err = ikev2.NewServiceClient(vppConn).Ikev2SetLocalKey(ctx, &ikev2.Ikev2SetLocalKey{
		KeyFile: privateKeyFileName,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetLocalKey returned error")
	}
	log.FromContext(ctx).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetLocalKey").Debug("completed")

	return nil
}

func setFQDN(ctx context.Context, vppConn api.Connection, mechanism *ipsec.Mechanism, profileName string, isClient bool) error {
	now := time.Now()

	// We need unique values per client/server. Using public keys
	fqdnLocal := mechanism.SrcPublicKey()[:64]
	fqdnRemote := mechanism.DstPublicKey()[:64]
	if !isClient {
		fqdnLocal, fqdnRemote = fqdnRemote, fqdnLocal
	}
	_, err := ikev2.NewServiceClient(vppConn).Ikev2ProfileSetID(ctx, &ikev2.Ikev2ProfileSetID{
		Name:    profileName,
		IsLocal: true,
		IDType:  2, // FQDN
		DataLen: uint32(len(fqdnLocal)),
		Data:    []byte(fqdnLocal),
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2ProfileSetID returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("IsLocal", "true").
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2ProfileSetID").Debug("completed")

	now = time.Now()
	_, err = ikev2.NewServiceClient(vppConn).Ikev2ProfileSetID(ctx, &ikev2.Ikev2ProfileSetID{
		Name:    profileName,
		IsLocal: false,
		IDType:  2, // FQDN
		DataLen: uint32(len(fqdnRemote)),
		Data:    []byte(fqdnRemote),
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2ProfileSetID returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("IsLocal", "false").
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2ProfileSetID").Debug("completed")
	return nil
}

func setTrafficSelector(ctx context.Context, vppConn api.Connection, profileName string, conn *networkservice.Connection, isClient bool) error {
	now := time.Now()
	for _, addr := range conn.GetContext().GetIpContext().GetSrcIpAddrs() {
		a, err := ip_types.ParseAddressWithPrefix(addr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse address with prefix %s", addr)
		}
		_, err = ikev2.NewServiceClient(vppConn).Ikev2ProfileSetTs(ctx, &ikev2.Ikev2ProfileSetTs{
			Name: profileName,
			Ts: ikev2_types.Ikev2Ts{
				IsLocal:   isClient,
				StartPort: 0,
				EndPort:   65535,
				StartAddr: a.Address,
				EndAddr:   a.Address,
			},
		})
		if err != nil {
			return errors.Wrap(err, "vppapi Ikev2ProfileSetTs returned error")
		}
		log.FromContext(ctx).
			WithField("Name", profileName).
			WithField("IsLocal", isClient).
			WithField("Address", addr).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "Ikev2ProfileSetTs").Debug("completed")
	}

	for _, addr := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
		a, err := ip_types.ParseAddressWithPrefix(addr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse address with prefix %s", addr)
		}
		_, err = ikev2.NewServiceClient(vppConn).Ikev2ProfileSetTs(ctx, &ikev2.Ikev2ProfileSetTs{
			Name: profileName,
			Ts: ikev2_types.Ikev2Ts{
				IsLocal:   !isClient,
				StartPort: 0,
				EndPort:   65535,
				StartAddr: a.Address,
				EndAddr:   a.Address,
			},
		})
		if err != nil {
			return errors.Wrap(err, "vppapi Ikev2ProfileSetTs returned error")
		}
		log.FromContext(ctx).
			WithField("Name", profileName).
			WithField("IsLocal", !isClient).
			WithField("Address", addr).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "Ikev2ProfileSetTs").Debug("completed")
	}
	return nil
}

func protectTunnel(ctx context.Context, vppConn api.Connection, profileName string, tunSwIfIndex interface_types.InterfaceIndex) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2SetTunnelInterface(ctx, &ikev2.Ikev2SetTunnelInterface{
		Name:      profileName,
		SwIfIndex: tunSwIfIndex,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetTunnelInterface returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("SwIfIndex", tunSwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetTunnelInterface").Debug("completed")
	return nil
}

func setResponder(ctx context.Context, vppConn api.Connection, profileName string, hostSwIfIndex interface_types.InterfaceIndex, responderIP net.IP) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2SetResponder(ctx, &ikev2.Ikev2SetResponder{
		Name: profileName,
		Responder: ikev2_types.Ikev2Responder{
			SwIfIndex: hostSwIfIndex,
			Addr:      types.ToVppAddress(responderIP),
		},
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetResponder returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("SwIfIndex", hostSwIfIndex).
		WithField("Addr", responderIP.String()).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetResponder").Debug("completed")
	return nil
}

func setTransforms(ctx context.Context, vppConn api.Connection, profileName string) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2SetIkeTransforms(ctx, &ikev2.Ikev2SetIkeTransforms{
		Name: profileName,
		Tr: ikev2_types.Ikev2IkeTransforms{
			CryptoAlg:     12, // aes-cbc
			CryptoKeySize: 256,
			IntegAlg:      2,  // sha1-96
			DhGroup:       14, // modp-2048
		},
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetIkeTransforms returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetIkeTransforms").Debug("completed")

	now = time.Now()
	_, err = ikev2.NewServiceClient(vppConn).Ikev2SetEspTransforms(ctx, &ikev2.Ikev2SetEspTransforms{
		Name: profileName,
		Tr: ikev2_types.Ikev2EspTransforms{
			CryptoAlg:     12, // aes-cbc,
			CryptoKeySize: 256,
			IntegAlg:      2, // sha1-96
		},
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetEspTransforms returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetEspTransforms").Debug("completed")
	return nil
}

func setSaLifetime(ctx context.Context, vppConn api.Connection, profileName string) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2SetSaLifetime(ctx, &ikev2.Ikev2SetSaLifetime{
		Name:            profileName,
		Lifetime:        3600,
		LifetimeJitter:  10,
		Handover:        5,
		LifetimeMaxdata: 0,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2SetSaLifetime returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2SetSaLifetime").Debug("completed")
	return nil
}

func saInit(ctx context.Context, vppConn api.Connection, profileName string) error {
	now := time.Now()
	_, err := ikev2.NewServiceClient(vppConn).Ikev2InitiateSaInit(ctx, &ikev2.Ikev2InitiateSaInit{
		Name: profileName,
	})
	if err != nil {
		return errors.Wrap(err, "vppapi Ikev2InitiateSaInit returned error")
	}
	log.FromContext(ctx).
		WithField("Name", profileName).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "Ikev2InitiateSaInit").Debug("completed")
	return nil
}

func delInterface(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) {
	if mechanism := ipsec.ToMechanism(conn.GetMechanism()); mechanism != nil {
		profileName := fmt.Sprintf("%s-%s", isClientPrefix(isClient), conn.Id)
		_ = addDelProfile(ctx, vppConn, profileName, false)
		_ = delIPSecTunnel(ctx, vppConn, isClient)
	}
}

func generateRSAKey() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate private key")
	}

	return key, nil
}

func dumpPrivateKeyToFile(privatekey *rsa.PrivateKey, profileName string, isClient bool) (string, error) {
	dir := path.Join(os.TempDir(), profileName)
	err := os.Mkdir(dir, 0o700)
	if err != nil && !os.IsExist(err) {
		return "", errors.Wrapf(err, "failed to create directory %s", dir)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privatekey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	file, err := os.Create(path.Clean(path.Join(dir, isClientPrefix(isClient)+"-key.pem")))
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file %s", path.Clean(path.Join(dir, isClientPrefix(isClient)+"-key.pem")))
	}
	err = pem.Encode(file, privateKeyBlock)
	if err != nil {
		return "", errors.Wrap(err, "encode process has failed")
	}

	return file.Name(), nil
}

func createCertBase64(privatekey *rsa.PrivateKey, isClient bool) (string, error) {
	// Generate cryptographically strong pseudo-random between 0 - max
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", errors.Wrap(err, "failed to get random integer")
	}

	template := &x509.Certificate{
		SerialNumber: n,
		Subject: pkix.Name{
			CommonName: isClientPrefix(isClient),
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Minute * 5),
	}
	certbytes, err := x509.CreateCertificate(rand.Reader, template, template, &privatekey.PublicKey, privatekey)
	if err != nil {
		return "", errors.Wrap(err, "failed to create certificate")
	}
	return base64.StdEncoding.EncodeToString(certbytes), nil
}

func dumpCertBase64ToFile(base64key, profileName string, isClient bool) (string, error) {
	dir := path.Join(os.TempDir(), profileName)
	err := os.Mkdir(dir, 0o700)
	if err != nil && !os.IsExist(err) {
		return "", errors.Wrapf(err, "failed to create directory %s", dir)
	}

	certbytes, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode base64 encoded string %s", base64key)
	}

	publicKeyBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certbytes,
	}

	file, err := os.Create(path.Clean(path.Join(dir, isClientPrefix(!isClient)+"-cert.pem")))
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file %s", path.Clean(path.Join(dir, isClientPrefix(!isClient)+"-cert.pem")))
	}
	err = pem.Encode(file, publicKeyBlock)
	if err != nil {
		return "", errors.Wrap(err, "encode process has failed")
	}

	return file.Name(), nil
}

func isClientPrefix(isClient bool) string {
	if isClient {
		return "client"
	}
	return "server"
}
