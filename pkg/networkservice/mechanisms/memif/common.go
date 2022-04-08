// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

//+build linux

package memif

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/networkservicemesh/sdk-vpp/pkg/tools/dumptool"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

// NetNSInfo contains shared info for server and client
type NetNSInfo struct {
	netNS     netns.NsHandle
	netNSPath string
}

// NewNetNSInfo should be called only once for single chain
func newNetNSInfo() NetNSInfo {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fd, err := unix.Open("/proc/thread-self/ns/net", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		panic("failed to open '/proc/thread-self/ns/net': " + err.Error())
	}
	netNSPath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)

	netNS, err := netns.GetFromPath(netNSPath)
	if err != nil {
		panic("failed to get current net NS: " + err.Error())
	}

	return NetNSInfo{
		netNSPath: netNSPath,
		netNS:     netNS,
	}
}

func createMemifSocket(ctx context.Context, mechanism *memifMech.Mechanism, vppConn api.Connection, isClient bool, netNS netns.NsHandle) (socketID uint32, err error) {
	socketFilename, err := getVppSocketFilename(mechanism, netNS)
	if err != nil {
		return 0, err
	}

	memifSocketAddDel := &memif.MemifSocketFilenameAddDelV2{
		IsAdd:          true,
		SocketID:       ^uint32(0),
		SocketFilename: socketFilename,
	}

	now := time.Now()

	reply, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDelV2(ctx, memifSocketAddDel)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	memifSocketAddDel.SocketID = reply.SocketID

	log.FromContext(ctx).
		WithField("SocketID", memifSocketAddDel.SocketID).
		WithField("SocketFilename", memifSocketAddDel.SocketFilename).
		WithField("IsAdd", memifSocketAddDel.IsAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifSocketFilenameAddDel").Debug("completed")

	store(ctx, isClient, memifSocketAddDel)

	return memifSocketAddDel.SocketID, nil
}

func deleteMemifSocket(ctx context.Context, vppConn api.Connection, isClient bool) error {
	memifSocketAddDel, ok := load(ctx, isClient)
	if !ok {
		return nil
	}

	memifSocketAddDel.IsAdd = false

	now := time.Now()

	if _, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDelV2(ctx, memifSocketAddDel); err != nil {
		return errors.WithStack(err)
	}

	log.FromContext(ctx).
		WithField("SocketID", memifSocketAddDel.SocketID).
		WithField("SocketFilename", memifSocketAddDel.SocketFilename).
		WithField("IsAdd", memifSocketAddDel.IsAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifSocketFilenameAddDel").Debug("completed")

	return nil
}

func createMemif(ctx context.Context, vppConn api.Connection, socketID uint32, mode memif.MemifMode, isClient bool) error {
	role := memif.MEMIF_ROLE_API_MASTER
	if isClient {
		role = memif.MEMIF_ROLE_API_SLAVE
	}
	now := time.Now()
	memifCreate := &memif.MemifCreate{
		Role:     role,
		SocketID: socketID,
		Mode:     mode,
	}
	rsp, err := memif.NewServiceClient(vppConn).MemifCreate(ctx, memifCreate)
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", rsp.SwIfIndex).
		WithField("Role", memifCreate.Role).
		WithField("SocketID", memifCreate.SocketID).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifCreate").Debug("completed")
	ifindex.Store(ctx, isClient, rsp.SwIfIndex)

	now = time.Now()
	if _, err := interfaces.NewServiceClient(vppConn).SwInterfaceSetRxMode(ctx, &interfaces.SwInterfaceSetRxMode{
		SwIfIndex: rsp.SwIfIndex,
		Mode:      interface_types.RX_MODE_API_ADAPTIVE,
	}); err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", rsp.SwIfIndex).
		WithField("mode", interface_types.RX_MODE_API_ADAPTIVE).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "SwInterfaceSetRxMode").Debug("completed")

	if isClient {
		up.Store(ctx, isClient, true)
	}
	return nil
}

func deleteMemif(ctx context.Context, vppConn api.Connection, isClient bool) error {
	swIfIndex, ok := ifindex.LoadAndDelete(ctx, isClient)
	if !ok {
		return nil
	}
	now := time.Now()
	memifDel := &memif.MemifDelete{
		SwIfIndex: swIfIndex,
	}
	_, err := memif.NewServiceClient(vppConn).MemifDelete(ctx, memifDel)
	if err != nil {
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("swIfIndex", memifDel.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifDelete").Debug("completed")
	return nil
}

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, dumpMap *dumptool.Map, isClient bool, netNS netns.NsHandle) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if val, loaded := dumpMap.LoadAndDelete(conn.GetId()); loaded {
			ifindex.Store(ctx, isClient, val.(interface_types.InterfaceIndex))
		}

		if !isClient {
			mechanism.SetSocketFilename(socketFile(conn))
		}
		// This connection has already been created
		if _, ok := ifindex.Load(ctx, isClient); ok {
			socketFilename, err := getVppSocketFilename(mechanism, netNS)
			if err != nil {
				return err
			}
			if memifSocketAddDel, ok := load(ctx, isClient); ok && memifSocketAddDel.SocketFilename == socketFilename {
				return nil
			}
		}
		_ = del(ctx, conn, vppConn, dumpMap, isClient)

		mode := memif.MEMIF_MODE_API_IP
		if conn.GetPayload() == payload.Ethernet {
			mode = memif.MEMIF_MODE_API_ETHERNET
		}
		socketID, err := createMemifSocket(ctx, mechanism, vppConn, isClient, netNS)
		if err != nil {
			return err
		}
		if err := createMemif(ctx, vppConn, socketID, mode, isClient); err != nil {
			return err
		}
	}
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, dumpMap *dumptool.Map, isClient bool) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		if val, loaded := dumpMap.LoadAndDelete(conn.GetId()); loaded {
			ifindex.Store(ctx, isClient, val.(interface_types.InterfaceIndex))
		}

		if err := deleteMemif(ctx, vppConn, isClient); err != nil {
			return err
		}
		if err := deleteMemifSocket(ctx, vppConn, isClient); err != nil {
			return err
		}
	}
	return nil
}

func socketFile(conn *networkservice.Connection) string {
	return "@" + filepath.Join(os.TempDir(), "memif", conn.GetId(), "memif.socket")
}

func getVppSocketFilename(mechanism *memifMech.Mechanism, netNS netns.NsHandle) (string, error) {
	u, err := url.Parse(mechanism.GetNetNSURL())
	if err != nil {
		return "", errors.Wrapf(err, "not a valid url %s", mechanism.GetNetNSURL())
	}
	if u.Scheme != memifMech.FileScheme {
		return "", errors.Errorf("socket file url must have scheme %s, actual %s", memifMech.FileScheme, u.Scheme)
	}

	targetNetNS, err := netns.GetFromPath(u.Path)
	if err != nil {
		return "", err
	}
	defer func() { _ = targetNetNS.Close() }()

	if !targetNetNS.Equal(netNS) {
		return "@netns:" + u.Path + mechanism.GetSocketFilename(), nil
	}
	return mechanism.GetSocketFilename(), nil
}

func dump(ctx context.Context, vppConn api.Connection, podName string, timeout time.Duration, isClient bool) (*dumptool.Map, error) {
	return dumptool.DumpInterfaces(ctx, vppConn, podName, timeout, isClient,
		/* Function on dump */
		func(details *interfaces.SwInterfaceDetails) (interface{}, error) {
			if details.InterfaceDevType == dumptool.DevTypeMemif {
				return details.SwIfIndex, nil
			}
			return nil, errors.New("Doesn't match the memif interface")
		},
		/* Function on delete */
		func(ifindex interface{}) error {
			/* Find memif interface to determine its SocketID */
			memClient, err := memif.NewServiceClient(vppConn).MemifDump(ctx, &memif.MemifDump{})
			if err != nil {
				return err
			}
			defer func() { _ = memClient.Close() }()
			for {
				var memDetails *memif.MemifDetails
				memDetails, err = memClient.Recv()
				if err == io.EOF {
					break
				}
				if memDetails == nil || memDetails.SwIfIndex != ifindex {
					continue
				}

				_, err = memif.NewServiceClient(vppConn).MemifDelete(ctx, &memif.MemifDelete{
					SwIfIndex: ifindex.(interface_types.InterfaceIndex),
				})
				if err != nil {
					return err
				}

				/* Find memifSocketFilename by SocketID */
				var memSockClient memif.RPCService_MemifSocketFilenameDumpClient
				memSockClient, err = memif.NewServiceClient(vppConn).MemifSocketFilenameDump(ctx, &memif.MemifSocketFilenameDump{})
				if err != nil {
					return err
				}
				defer func() { _ = memSockClient.Close() }()
				for {
					var memSockDetails *memif.MemifSocketFilenameDetails
					memSockDetails, err = memSockClient.Recv()
					if err == io.EOF {
						break
					}
					if memSockDetails == nil || memSockDetails.SocketID != memDetails.SocketID {
						continue
					}
					_, err = memif.NewServiceClient(vppConn).MemifSocketFilenameAddDelV2(ctx, &memif.MemifSocketFilenameAddDelV2{
						IsAdd:          false,
						SocketID:       memSockDetails.SocketID,
						SocketFilename: memSockDetails.SocketFilename,
					})
					if err != nil {
						return err
					}
					break
				}
				break
			}
			return err
		})
}
