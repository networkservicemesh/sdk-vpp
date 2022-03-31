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

package dumptool

import (
	"context"
	"io"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/pkg/errors"
)

// DumpFn -
type DumpFn func(details *interfaces.SwInterfaceDetails) (interface{}, error)

// DeleteFn -
type DeleteFn func(value interface{}) error

// DumpOption -
type DumpOption struct {
	PodName string
	Timeout time.Duration
}

// DumpInterfaces - key - connectionID, value - value
func DumpInterfaces(ctx context.Context, vppConn api.Connection, podName string, timeout time.Duration, isClient bool, onDump DumpFn, onDelete DeleteFn) (*Map, error) {
	client, err := interfaces.NewServiceClient(vppConn).SwInterfaceDump(ctx, &interfaces.SwInterfaceDump{})
	if err != nil {
		return nil, errors.Wrap(err, "SwInterfaceDump error")
	}
	defer func() { _ = client.Close() }()

	retMap := NewMap(ctx, timeout)
	for {
		details, err := client.Recv()
		if err == io.EOF || details == nil {
			break
		}

		t, err := ConvertFromTag(details.Tag)
		if err != nil {
			continue
		}
		if t.PodName != podName || t.IsClient != isClient {
			continue
		}

		if val, err := onDump(details); err == nil && val != nil {
			retMap.Store(t.ConnID, val, onDelete)
		}
	}
	return retMap, nil
}
