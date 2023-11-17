// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
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

//go:build linux
// +build linux

package stats

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"go.fd.io/govpp/adapter"
	"go.fd.io/govpp/adapter/statsclient"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/core"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	interfaces "github.com/networkservicemesh/govpp/binapi/interface"
	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

type interfacesInfo struct {
	interfaceName    string
	interfaceFwdName string
	interfaceType    string
}

func (i *interfacesInfo) getInterfaceDetails() string {
	return combineDetails(i.interfaceType, i.interfaceName)
}

func (i *interfacesInfo) getInterfaceFwdDetails() string {
	return combineDetails(i.interfaceType, i.interfaceFwdName)
}

// Save retrieved vpp interface metrics in pathSegment
func retrieveMetrics(ctx context.Context, statsConn *core.StatsConnection, vppConn api.Connection, conn *networkservice.Connection, isClient bool) {
	segment := conn.Path.PathSegments[conn.Path.Index]

	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return
	}
	stats := new(api.InterfaceStats)
	if err := statsConn.GetInterfaceStats(stats); err != nil {
		log.FromContext(ctx).Errorf("getting interface stats failed:", err)
		return
	}

	info, err := getInterfacesInfo(ctx, vppConn, conn, swIfIndex)
	if err != nil {
		log.FromContext(ctx).Errorf("getting interfaces info failed:", err)
		return
	}

	addName := "server_"
	if isClient {
		addName = "client_"
	}
	for idx := range stats.Interfaces {
		iface := &stats.Interfaces[idx]
		if iface.InterfaceIndex != uint32(swIfIndex) {
			continue
		}

		if segment.Metrics == nil {
			segment.Metrics = make(map[string]string)
		}
		segment.Metrics[addName+"rx_bytes"] = strconv.FormatUint(iface.Rx.Bytes, 10)
		segment.Metrics[addName+"tx_bytes"] = strconv.FormatUint(iface.Tx.Bytes, 10)
		segment.Metrics[addName+"rx_packets"] = strconv.FormatUint(iface.Rx.Packets, 10)
		segment.Metrics[addName+"tx_packets"] = strconv.FormatUint(iface.Tx.Packets, 10)
		segment.Metrics[addName+"interface"] = info.getInterfaceDetails()
		segment.Metrics[addName+"interface_fwd"] = info.getInterfaceFwdDetails()
		segment.Metrics[addName+"drops"] = strconv.FormatUint(iface.Drops, 10)
		break
	}
}

func initFunc(chainCtx context.Context, statsSocket string) (*core.StatsConnection, error) {
	if statsSocket == "" {
		statsSocket = adapter.DefaultStatsSocket
	}
	statsConn, err := core.ConnectStats(statsclient.NewStatsClient(statsSocket))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Stats API")
	}
	go func() {
		<-chainCtx.Done()
		statsConn.Disconnect()
	}()
	return statsConn, nil
}

func getInterfacesInfo(ctx context.Context, vppConn api.Connection, conn *networkservice.Connection, swIfIndex interface_types.InterfaceIndex) (*interfacesInfo, error) {
	info := &interfacesInfo{
		interfaceName: conn.Mechanism.Parameters["name"],
	}

	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{
		SwIfIndex: swIfIndex,
	})

	if err != nil {
		return nil, err
	}

	for {
		details, err := client.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		info.interfaceFwdName = details.InterfaceName
		info.interfaceType = details.InterfaceDevType
	}

	return info, nil
}

func combineDetails(iType, iName string) string {
	return fmt.Sprintf("%s/%s", iType, iName)
}
