// Copyright (c) 2022 Cisco and/or its affiliates.
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

package nsmonitor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	monitorInterval = 250 * time.Millisecond
)

type monitorItem struct {
	pause   string
	clients map[string]chan bool
}

type netNSMonitor struct {
	chainCtx context.Context
	executor serialize.Executor
	nses     map[uint64]monitorItem
	on       bool
	stopCh   chan struct{}
}

func newMonitor(chainCtx context.Context) Monitor {
	return &netNSMonitor{
		chainCtx: chainCtx,
		executor: serialize.Executor{},
		nses:     map[uint64]monitorItem{},
		stopCh:   make(chan struct{}, 1),
	}
}

func (m *netNSMonitor) Subscribe(ctx context.Context, inodeURL, connID string) <-chan bool {
	ch := make(chan bool, 1)

	go func() {
		logger := log.FromContext(ctx).WithField("component", "netNsMonitor").WithField("inodeURL", inodeURL)

		inode, pause, err := getTarget(inodeURL)
		if err != nil {
			logger.Error(err.Error())
			ch <- false
			return
		}

		m.executor.AsyncExec(func() {
			if _, ok := m.nses[inode]; !ok {
				m.nses[inode] = monitorItem{
					pause:   pause,
					clients: map[string]chan bool{},
				}
			}

			m.nses[inode].clients[connID] = ch

			if !m.on {
				m.on = true
				go m.monitor()
			}
		})
	}()

	return ch
}

func (m *netNSMonitor) Unsubscribe(ctx context.Context, connID string) {
	m.executor.AsyncExec(func() {
		for inode, item := range m.nses {
			if ch, ok := item.clients[connID]; ok {
				ch <- false
				delete(item.clients, connID)
			}
			if len(item.clients) == 0 {
				delete(m.nses, inode)
			}
		}
		if len(m.nses) == 0 {
			m.on = false
			m.stopCh <- struct{}{}
		}
	})
}

func (m *netNSMonitor) monitor() {
	for {
		select {
		case <-time.After(monitorInterval):
			m.checkSubscribedNS()
		case <-m.stopCh:
			return
		case <-m.chainCtx.Done():
			return
		}
	}
}

func (m *netNSMonitor) checkSubscribedNS() {
	m.executor.AsyncExec(func() {
		for inode, item := range m.nses {
			if _, err := os.Stat(item.pause); err != nil {
				println("netNsMonitor", fmt.Sprintf("%v died", inode))
				for _, ch := range item.clients {
					ch <- true
				}
			}
		}
	})
}
