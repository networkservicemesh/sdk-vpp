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

// Package dumptool provides utilities for vpp interfaces dump
package dumptool

import (
	"context"
	"io"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/tagtool"
)

// DumpFn - action with dumped NSM interfaces
type DumpFn func(details *interfaces.SwInterfaceDetails) error

// DumpOption - option that configures chain elements
type DumpOption struct {
	Ctx     context.Context
	PodName string
}

// DumpVppInterfaces - dumps vpp interfaces by tag.
//	- onDump - determines what to do if we found an NSM interface during the dump
func DumpVppInterfaces(ctx context.Context, vppConn api.Connection, podName string, isClient bool, onDump DumpFn) error {
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if err != nil {
		return errors.Wrap(err, "SwInterfaceDump error")
	}
	defer func() { _ = client.Close() }()

	for {
		details, err := client.Recv()
		if err == io.EOF || details == nil {
			break
		}

		t, err := tagtool.ConvertFromString(details.Tag)
		if err != nil {
			continue
		}
		if t.PodName != podName || t.IsClient != isClient {
			continue
		}

		if err := onDump(details); err != nil {
			log.FromContext(ctx).Error(err)
		}
	}
	return nil
}
