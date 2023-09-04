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

package unnumbered

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/govpp/binapi/interface_types"
	"go.fd.io/govpp/api"

	interfaces "github.com/networkservicemesh/govpp/binapi/interface"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func addDel(ctx context.Context, vppConn api.Connection, isClient, isAdd bool, loadIfaceFn func(ctx context.Context, isClient bool) (value interface_types.InterfaceIndex, ok bool)) error {
	if _, ok := load(ctx, isClient); ok {
		return nil
	}

	swIfIndex, ok := ifindex.Load(ctx, isClient)
	if !ok {
		return errors.New("no swIfIndex available")
	}
	loopIfIndex, ok := loadIfaceFn(ctx, isClient)
	if !ok {
		return errors.New("no loopback available")
	}

	now := time.Now()
	if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetUnnumbered(ctx, &interfaces.SwInterfaceSetUnnumbered{
		SwIfIndex:           loopIfIndex,
		UnnumberedSwIfIndex: swIfIndex,
		IsAdd:               isAdd,
	}); err != nil {
		return errors.Wrap(err, "vppapi SwInterfaceSetUnnumbered returned error")
	}
	log.FromContext(ctx).
		WithField("swIfIndex", loopIfIndex).
		WithField("unnumberedSwIfIndex", swIfIndex).
		WithField("isAdd", isAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceSetUnnumbered").Debug("completed")

	store(ctx, isClient)

	return nil
}
