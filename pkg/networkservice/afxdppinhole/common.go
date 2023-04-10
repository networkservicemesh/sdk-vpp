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

package afxdppinhole

import (
	"context"
	"strconv"

	"github.com/cilium/ebpf"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/pinhole"
)

type bpfObjects struct {
	XdpPinhole *ebpf.Map `ebpf:"nsm_xdp_pinhole"`
}

func fromMechanism(mechanism *networkservice.Mechanism, isClient bool) uint16 {
	if mechanism.GetParameters() == nil {
		return 0
	}
	portKey := common.DstPort
	if isClient {
		portKey = common.SrcPort
	}
	portStr, ok := mechanism.GetParameters()[portKey]
	if !ok {
		return 0
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return 0
	}
	return uint16(port)
}

func fromContext(ctx context.Context, isClient bool) uint16 {
	v, ok := pinhole.LoadExtra(ctx, isClient)
	if !ok {
		return 0
	}

	return v.Port()
}

func updateXdpPinhole(keys []uint16, elfPath, bpfFSDir string) error {
	for _, k := range keys {
		if k == 0 {
			continue
		}
		key := k
		collectionSpec, err := ebpf.LoadCollectionSpec(elfPath)
		if err != nil {
			return err
		}
		objs := bpfObjects{}
		err = collectionSpec.LoadAndAssign(&objs, &ebpf.CollectionOptions{
			Maps: ebpf.MapOptions{
				PinPath: bpfFSDir,
			},
		})
		if err != nil {
			return err
		}

		var val uint16
		err = objs.XdpPinhole.Update(&key, &val, ebpf.UpdateAny)
		if err != nil {
			return err
		}
	}
	return nil
}
