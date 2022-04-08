// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// +build linux

package memif_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_MemifClient_ShouldAppendMechanismIfMemifMechanismMissed(t *testing.T) {
	c := chain.NewNetworkServiceClient(metadata.NewClient(), memif.NewClient(context.Background(), nil))

	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{},
	}

	_, err := c.Request(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, req.MechanismPreferences, 1)

	req.MechanismPreferences = []*networkservice.Mechanism{
		{
			Type: kernel.MECHANISM,
		},
	}

	_, err = c.Request(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, req.MechanismPreferences, 2)
}

func Test_MemifClient_ShouldNotDuplicateMechanisms(t *testing.T) {
	c := chain.NewNetworkServiceClient(metadata.NewClient(), memif.NewClient(context.Background(), nil))

	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: memif.MECHANISM,
			},
		},
	}

	for i := 0; i < 10; i++ {
		_, err := c.Request(context.Background(), req)
		require.NoError(t, err)
	}

	require.Len(t, req.MechanismPreferences, 1)
}
