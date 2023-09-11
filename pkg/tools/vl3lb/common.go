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

package vl3lb

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/edwarnicke/serialize"
	"go.fd.io/govpp/api"

	"github.com/networkservicemesh/govpp/binapi/cnat"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/govpp/binapi/ip_types"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/types"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// Endpoint contains the main fields for the VPP plugin
type Endpoint struct {
	IP   net.IP
	Port uint16
}

// Equals returns true if Endpoints are equal
func (e *Endpoint) Equals(endpoint *Endpoint) bool {
	return e.IP.Equal(endpoint.IP) && e.Port == endpoint.Port
}

// Handler works with load balancer servers. It is based on CNAT VPP-plugin
type Handler struct {
	vppConn    api.Connection
	lbEndpoint cnat.CnatEndpoint
	proto      ip_types.IPProto
	isRealIP   uint8
	lbType     cnat.CnatLbType

	// [vl3-NSE] --> [connID]*Endpoint
	// We store it this way because the plugin does not add, but only updates existing entries. Therefore, to add/delete one entry, we must also pass the old ones.
	servers  map[string]map[string]*Endpoint
	executor serialize.Executor
}

// NewHandler creates a Handler.
// endpoint contains Load Balancer parameters. The clients can reach the LB with endpoint.Addr:endpoint:Port
func NewHandler(vppConn api.Connection, endpoint *Endpoint, proto ip_types.IPProto) *Handler {
	return &Handler{
		vppConn: vppConn,
		lbEndpoint: cnat.CnatEndpoint{
			Addr:      types.ToVppAddress(endpoint.IP),
			SwIfIndex: interface_types.InterfaceIndex(^uint32(0)),
			Port:      endpoint.Port,
		},
		proto:    proto,
		isRealIP: 1,
		lbType:   cnat.CNAT_LB_TYPE_MAGLEV,
		servers:  make(map[string]map[string]*Endpoint),
	}
}

func cnatTranslationString(c *cnat.CnatTranslation) string {
	str := fmt.Sprintf("%s:%d", c.Vip.Addr.String(), c.Vip.Port)
	for _, p := range c.Paths {
		str = fmt.Sprintf("%s to %s -> %s:%d, ", str, p.SrcEp.Addr, p.DstEp.Addr, p.DstEp.Port)
	}
	return str
}

// AddServers adds the real servers to the VPP plugin
func (c *Handler) AddServers(ctx context.Context, vl3NSEName string, add map[string]*Endpoint) error {
	var err error
	<-c.executor.AsyncExec(func() {
		if _, ok := c.servers[vl3NSEName]; !ok {
			c.servers[vl3NSEName] = make(map[string]*Endpoint)
		}

		updateRequired := false
		for k, v := range add {
			if endpoint, ok := c.servers[vl3NSEName][k]; !ok || !endpoint.Equals(v) {
				c.servers[vl3NSEName][k] = v
				updateRequired = true
			}
		}

		if updateRequired {
			err = c.updateVPPCnat(ctx)
		}
	})

	return err
}

// DeleteServers deletes the real servers from the VPP plugin
func (c *Handler) DeleteServers(ctx context.Context, vl3NSEName string, del []string) error {
	var err error
	<-c.executor.AsyncExec(func() {
		if _, ok := c.servers[vl3NSEName]; !ok {
			return
		}

		updateRequired := false
		for _, id := range del {
			delete(c.servers[vl3NSEName], id)
			updateRequired = true
		}
		if len(c.servers[vl3NSEName]) == 0 {
			log.FromContext(ctx).WithField("vl3Loadbalancer", "DeleteServers").Infof("Delete VL3NSE: %s ", vl3NSEName)
			delete(c.servers, vl3NSEName)
		}

		if updateRequired {
			err = c.updateVPPCnat(ctx)
		}
	})

	return err
}

// GetServerIDsByVL3Name returns the list of the servers belonging to the vl3-NSE
func (c *Handler) GetServerIDsByVL3Name(vl3NSEName string) []string {
	var list []string
	<-c.executor.AsyncExec(func() {
		for id := range c.servers[vl3NSEName] {
			list = append(list, id)
		}
	})
	return list
}

// should work under executor
func (c *Handler) updateVPPCnat(ctx context.Context) error {
	var paths []cnat.CnatEndpointTuple
	for _, vl3Servers := range c.servers {
		for _, s := range vl3Servers {
			paths = append(paths, cnat.CnatEndpointTuple{
				DstEp: cnat.CnatEndpoint{
					Addr:      types.ToVppAddress(s.IP),
					SwIfIndex: interface_types.InterfaceIndex(^uint32(0)),
					Port:      s.Port,
				},
				SrcEp: cnat.CnatEndpoint{
					Addr:      c.lbEndpoint.Addr,
					SwIfIndex: interface_types.InterfaceIndex(^uint32(0)),
				},
			})
		}
	}
	if len(paths) == 0 {
		now := time.Now()
		cnatTranslationDel := cnat.CnatTranslationDel{ID: 0}
		_, err := cnat.NewServiceClient(c.vppConn).CnatTranslationDel(ctx, &cnatTranslationDel)
		if err != nil {
			return err
		}

		log.FromContext(ctx).
			WithField("translationID", cnatTranslationDel.ID).
			WithField("duration", time.Since(now)).
			WithField("vppapi", "CnatTranslationDel").Debug("completed")
		return nil
	}

	now := time.Now()
	cnatTranslation := cnat.CnatTranslation{
		Vip:      c.lbEndpoint,
		ID:       0,
		IPProto:  c.proto,
		IsRealIP: c.isRealIP,
		LbType:   c.lbType,
		NPaths:   uint32(len(paths)),
		Paths:    paths,
	}
	_, err := cnat.NewServiceClient(c.vppConn).CnatTranslationUpdate(ctx, &cnat.CnatTranslationUpdate{Translation: cnatTranslation})
	if err != nil {
		return err
	}

	log.FromContext(ctx).
		WithField("translation", cnatTranslationString(&cnatTranslation)).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "CnatTranslationUpdate").Debug("completed")
	return nil
}
