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
	"os"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type netNSMonitor struct {
	chainCtx context.Context
	interval time.Duration
}

func newMonitor(chainCtx context.Context) Monitor {
	return &netNSMonitor{
		chainCtx: chainCtx,
		interval: 250 * time.Millisecond,
	}
}

func (m *netNSMonitor) Watch(ctx context.Context, inodeURL string) <-chan struct{} {
	result := make(chan struct{}, 1)
	logger := log.FromContext(ctx).WithField("component", "netNsMonitor").WithField("inodeURL", inodeURL)

	go func() {
		proc, err := resolveProcByInodeURL(inodeURL)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		defer func() {
			logger.Infof("stopping...")
			close(result)
		}()

		logger.Info("started")

		for ctx.Err() == nil {
			if _, err := os.Stat(proc); err != nil {
				logger.Error(err.Error())
				return
			}
			<-time.After(m.interval)
		}
	}()

	return result
}
