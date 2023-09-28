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
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// This function balances the load between the 'real' servers.
// The 'real' servers are searched by 'Selector'
//
// Briefly, the work looks like this:
// 1. Find all vl3-NSE
// 2. Connect to grpc monitor services of these vl3-NSE
// 3. Monitor all connections containing the vl3-NSE name
// 4. Filter out those connections that contain 'Selector' labels
// 5. Get SrcIPs from these connections
// 6. Configure VPP load balancing

func (lb *vl3lbClient) balanceService(ctx context.Context, conn *networkservice.Connection) {
	loggerLb := log.FromContext(ctx).WithField("LoadBalancer", conn.NetworkService)

	// 1. Find all vl3-NSE
	nseStream, err := lb.nseRegistryClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{conn.NetworkService},
		},
		Watch: true,
	})
	if err != nil {
		loggerLb.Errorf("error getting nses: %+v", err)
		return
	}

	lbVpp := newHandler(lb.vppConn, &endpoint{
		IP:   conn.GetContext().GetIpContext().GetSrcIPNets()[0].IP,
		Port: lb.port,
	}, lb.protocol)

	monitoredNSEs := make(map[string]string)
	for {
		msg, err := nseStream.Recv()
		if err != nil {
			break
		}

		// Do not monitor the same NSE multiple times
		if msg.Deleted {
			delete(monitoredNSEs, msg.GetNetworkServiceEndpoint().GetName())
			continue
		}
		if _, ok := monitoredNSEs[msg.GetNetworkServiceEndpoint().GetName()]; ok {
			continue
		}
		monitoredNSEs[msg.GetNetworkServiceEndpoint().GetName()] = ""

		go lb.balanceNSE(ctx, loggerLb, lbVpp, msg.NetworkServiceEndpoint)
	}
}

func (lb *vl3lbClient) balanceNSE(ctx context.Context, loggerLb log.Logger, lbVpp *handler, nse *registry.NetworkServiceEndpoint) {
	logger := loggerLb.WithField("NSE", nse.Name)
	urlNSE, err := url.Parse(nse.Url)
	if err != nil {
		logger.Errorf("url.Parse: %+v", err)
		return
	}

	// 2. Connect to grpc monitor services of these vl3-NSE
	dialCtx, cancelDial := context.WithTimeout(ctx, lb.dialTimeout)
	defer cancelDial()

	ccMonitor, err := grpc.DialContext(dialCtx, grpcutils.URLToTarget(urlNSE), lb.dialOpts...)
	if err != nil {
		logger.Errorf("failed to dial: %v, URL: %v, err: %v", nse.Name, urlNSE.String(), err.Error())
		return
	}
	logger.Debug("connected")

	// 3. Monitor all connections containing the vl3-NSE name
	monitorClientNse := networkservice.NewMonitorConnectionClient(ccMonitor)
	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()

	stream, err := monitorClientNse.MonitorConnections(monitorCtx, &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{{
			Name: nse.Name,
		}},
	})
	if err != nil {
		logger.WithField("NSE", nse.Name).Errorf("failed to MonitorConnections: %v", err.Error())
		return
	}
	for {
		event, err := stream.Recv()
		if err != nil {
			logger.WithField("NSE", nse.Name).Errorf("error MonitorConnections stream: %v", err.Error())
			_ = lbVpp.deleteServers(context.Background(), nse.Name, lbVpp.getServerIDsByVL3Name(nse.Name))
			break
		}

		// 4. Filter out those connections that contain 'Selector' labels
		add, del := filterConnections(event, lb.selector, lb.targetPort)
		// 6. Configure VPP load balancing
		if err = lbVpp.addServers(ctx, nse.Name, add); err != nil {
			logger.Errorf("addServers error: %v", err.Error())
		}
		if err = lbVpp.deleteServers(ctx, nse.Name, del); err != nil {
			logger.Errorf("deleteServers error: %v", err.Error())
		}
	}
}

func filterConnections(event *networkservice.ConnectionEvent, selector map[string]string, targetPort uint16) (add map[string]*endpoint, del []string) {
	add = make(map[string]*endpoint)
	for _, eventConnection := range event.Connections {
		for k, v := range selector {
			if eventConnection.Labels[k] == v {
				if event.GetType() == networkservice.ConnectionEventType_DELETE {
					del = append(del, eventConnection.Id)
				} else {
					// 5. Get SrcIPs from these connections
					switch eventConnection.GetState() {
					case networkservice.State_DOWN:
						del = append(del, eventConnection.Id)
					default:
						add[eventConnection.Id] = &endpoint{
							IP:   eventConnection.GetContext().GetIpContext().GetSrcIPNets()[0].IP,
							Port: targetPort,
						}
					}
					break
				}
			}
		}
	}
	return
}
