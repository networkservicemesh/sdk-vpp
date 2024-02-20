// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

// Package heal contains an implementation of LivenessChecker which uses VPP ping
package heal

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/govpp/binapi/ip_types"
	"github.com/networkservicemesh/govpp/binapi/ping"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"go.fd.io/govpp/api"
)

const (
	defaultTimeout = time.Second
	packetCount    = 4
	intervalFactor = 0.85
)

func waitForResponses(responseCh <-chan bool) bool {
	respCount := cap(responseCh)
	success := true
	for {
		resp, ok := <-responseCh
		if !ok {
			return false
		}
		if !resp {
			success = false
		}
		respCount--
		if respCount == 0 {
			return success
		}
	}
}

func doPing(
	deadlineCtx context.Context,
	vppConn api.Connection,
	srcIP, dstIP ip_types.Address,
	interval float64,
	repeat uint32,
	responseCh chan bool) {
	logger := log.FromContext(deadlineCtx).WithField("srcIP", srcIP.String()).WithField("dstIP", dstIP.String())

	if _, err := ping.NewServiceClient(vppConn).WantPingFinishedEvents(deadlineCtx, &ping.WantPingFinishedEvents{
		Address:  dstIP,
		Interval: interval,
		Repeat:   repeat,
	}); err != nil {
		logger.Error(errors.Wrap(err, "vppapi WantPingEvents returned error"))
		responseCh <- true
		return
	}

	watcher, err := vppConn.WatchEvent(deadlineCtx, &ping.PingFinishedEvent{})
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to watch ping.PingFinishedEvent").Error())
		responseCh <- true
		return
	}

	defer func() { watcher.Close() }()

	select {
	case <-deadlineCtx.Done():
		responseCh <- true
		return
	case rawMsg := <-watcher.Events():
		if msg, ok := rawMsg.(*ping.PingFinishedEvent); ok {
			if msg.ReplyCount == 0 {
				logger.Errorf("No packets received")
				responseCh <- false
				return
			}
			responseCh <- true
		}
	}
}

// VPPLivenessCheck return a liveness check function which uses VPP ping to check VPP dataplane
func VPPLivenessCheck(vppConn api.Connection) func(deadlineCtx context.Context, conn *networkservice.Connection) bool {
	return func(deadlineCtx context.Context, conn *networkservice.Connection) bool {
		deadline, ok := deadlineCtx.Deadline()
		if !ok {
			deadline = time.Now().Add(defaultTimeout)
		}
		timeout := time.Until(deadline)
		interval := timeout.Seconds() / float64(packetCount) * intervalFactor
		ipContext := conn.GetContext().GetIpContext()

		// Parse all source ips
		srcIPs := make([]ip_types.Address, 0)
		for _, srcIPNet := range ipContext.GetSrcIPNets() {
			srcIP, err := ip_types.ParseAddress(srcIPNet.IP.String())
			if err != nil {
				log.FromContext(deadlineCtx).Warnf("%v is not a valid source IPv4 or IPv6 address. Error: %s", srcIPNet.IP.String(), err.Error())
				continue
			}
			srcIPs = append(srcIPs, srcIP)
		}

		// Parse all destination ips
		dstIPs := make([]ip_types.Address, 0)
		for _, dstIPNet := range ipContext.GetDstIPNets() {
			dstIP, err := ip_types.ParseAddress(dstIPNet.IP.String())
			if err != nil {
				log.FromContext(deadlineCtx).Warnf("%v is not a valid destinaion IPv4 or IPv6 address. Error: %s", dstIPNet.IP.String(), err.Error())
				continue
			}
			dstIPs = append(dstIPs, dstIP)
		}

		combinationCount := len(srcIPs) * len(dstIPs)
		if combinationCount == 0 {
			log.FromContext(deadlineCtx).Debug("No IP address")
			return true
		}

		responseCh := make(chan bool, combinationCount)
		defer close(responseCh)
		for _, srcIP := range srcIPs {
			for _, dstIP := range dstIPs {
				go doPing(deadlineCtx, vppConn, srcIP, dstIP, interval, packetCount, responseCh)
			}
		}

		// Waiting for all ping results. If at least one fails - return false
		return waitForResponses(responseCh)
	}
}
