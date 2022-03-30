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

package nsmonitor_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/nsmonitor"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	testInode1 = "inode://4/4026534206"
	testInode2 = "inode://4/4026534149"
)

type testMonitor struct {
	inodes                     []string
	watchShouldCloseMonitoing  bool
	watchShouldCloseConnection bool
}

func (m *testMonitor) Watch(ctx context.Context, inodeURL string) <-chan struct{} {
	m.inodes = append(m.inodes, inodeURL)

	result := make(chan struct{}, 1)
	if m.watchShouldCloseConnection {
		result <- struct{}{}
		close(result)
	} else if m.watchShouldCloseMonitoing {
		close(result)
	}
	return result
}

func Test_Client_DontFailWhenNoInode(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
	)

	// no inodeURL in parameters
	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{},
			},
		},
	})
	require.NoError(t, err)
	require.Empty(t, monitor.inodes)

	// no mechanism parameters
	_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:        uuid.New().String(),
			Mechanism: &networkservice.Mechanism{},
		},
	})
	require.NoError(t, err)
	require.Empty(t, monitor.inodes)

	// no mechanism
	_, err = client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
		},
	})
	require.NoError(t, err)
	require.Empty(t, monitor.inodes)
}

func Test_Client_MonitorOnceSameConnectionId(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 1, len(monitor.inodes))
	require.Equal(t, testInode1, monitor.inodes[0])

	_, err = client.Request(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 1, len(monitor.inodes))
	require.Equal(t, testInode1, monitor.inodes[0])
}

func Test_Client_MonitorDifferentConnectionIds(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
	)

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request1)
	require.NoError(t, err)
	require.Equal(t, monitor.inodes, []string{testInode1})

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode2,
				},
			},
		},
	}

	_, err = client.Request(ctx, request2)
	require.NoError(t, err)
	require.Equal(t, monitor.inodes, []string{testInode1, testInode2})
}

func Test_Client_CloseMustCloseMonitoringGoroutine(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request)
	require.NoError(t, err)

	_, err = client.Close(ctx, request.Connection)
	require.NoError(t, err)

	goleak.VerifyNone(t)
}

func Test_Client_MonitorMustCloseMonitoringGoroutine(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{
		watchShouldCloseMonitoing: true,
	}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	counter := new(count.Client)

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
		counter,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request)
	require.NoError(t, err)

	// Connection is not closed, only monitoring stopped
	goleak.VerifyNone(t)
	require.Equal(t, 0, counter.Closes())
}

func Test_Client_MonitorCanCloseConnection(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{
		watchShouldCloseConnection: true,
	}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	counter := new(count.Client)

	client := chain.NewNetworkServiceClient(
		begin.NewClient(),
		metadata.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
		counter,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request)
	require.NoError(t, err)

	// we need some time for Close to finish
	<-time.After(200 * time.Millisecond)
	require.Equal(t, 1, counter.Closes())
}

func Test_Client_ChainContextMustCloseMonitoringGoroutine(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	monitor := testMonitor{}
	supplyMonitor := func(c context.Context) nsmonitor.Monitor { return &monitor }

	client := chain.NewNetworkServiceClient(
		metadata.NewClient(),
		begin.NewClient(),
		nsmonitor.NewClient(ctx, nsmonitor.WithSupplyMonitor(supplyMonitor)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.New().String(),
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{
					common.InodeURL: testInode1,
				},
			},
		},
	}

	_, err := client.Request(ctx, request)
	require.NoError(t, err)

	cancel()
	goleak.VerifyNone(t)
}
