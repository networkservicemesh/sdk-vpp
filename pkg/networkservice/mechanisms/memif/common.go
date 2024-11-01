// Copyright (c) 2020-2024 Cisco and/or its affiliates.
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

package memif

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/govpp/binapi/memif"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
	"go.fd.io/govpp/api"
	"golang.org/x/sys/unix"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/mechutils"
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
	newCtx := mechutils.ToSafeContext(ctx)
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

	reply, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDelV2(newCtx, memifSocketAddDel)
	if err != nil {
		return 0, errors.Wrap(err, "vppapi MemifSocketFilenameAddDel returned error")
	}
	memifSocketAddDel.SocketID = reply.SocketID

	log.FromContext(newCtx).
		WithField("SocketID", memifSocketAddDel.SocketID).
		WithField("SocketFilename", memifSocketAddDel.SocketFilename).
		WithField("IsAdd", memifSocketAddDel.IsAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifSocketFilenameAddDel").Debug("completed")

	store(newCtx, isClient, memifSocketAddDel)

	return memifSocketAddDel.SocketID, nil
}

func deleteMemifSocket(ctx context.Context, vppConn api.Connection, isClient bool) error {
	newCtx := mechutils.ToSafeContext(ctx)
	memifSocketAddDel, ok := load(newCtx, isClient)
	if !ok {
		return nil
	}

	memifSocketAddDel.IsAdd = false

	now := time.Now()

	if _, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDelV2(newCtx, memifSocketAddDel); err != nil {
		return errors.Wrap(err, "vppapi MemifSocketFilenameAddDelV2 returned error")
	}

	log.FromContext(newCtx).
		WithField("SocketID", memifSocketAddDel.SocketID).
		WithField("SocketFilename", memifSocketAddDel.SocketFilename).
		WithField("IsAdd", memifSocketAddDel.IsAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifSocketFilenameAddDel").Debug("completed")

	return nil
}

func createMemif(ctx context.Context, vppConn api.Connection, socketID uint32, mode memif.MemifMode, isClient bool) error {
	newCtx := mechutils.ToSafeContext(ctx)
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
	rsp, err := memif.NewServiceClient(vppConn).MemifCreate(newCtx, memifCreate)
	if err != nil {
		return errors.Wrap(err, "vppapi MemifCreate returned error")
	}
	log.FromContext(newCtx).
		WithField("swIfIndex", rsp.SwIfIndex).
		WithField("Role", memifCreate.Role).
		WithField("SocketID", memifCreate.SocketID).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifCreate").Debug("completed")
	ifindex.Store(newCtx, isClient, rsp.SwIfIndex)

	if isClient {
		up.Store(newCtx, isClient, true)
	}
	return nil
}

func deleteMemif(ctx context.Context, vppConn api.Connection, isClient bool) error {
	newCtx := mechutils.ToSafeContext(ctx)
	swIfIndex, ok := ifindex.LoadAndDelete(newCtx, isClient)
	if !ok {
		return nil
	}
	now := time.Now()
	memifDel := &memif.MemifDelete{
		SwIfIndex: swIfIndex,
	}
	_, err := memif.NewServiceClient(vppConn).MemifDelete(newCtx, memifDel)
	if err != nil {
		return errors.Wrap(err, "vppapi MemifDelete returned error")
	}
	log.FromContext(newCtx).
		WithField("swIfIndex", memifDel.SwIfIndex).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifDelete").Debug("completed")
	return nil
}

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool, netNS netns.NsHandle) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
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
		_ = del(ctx, conn, vppConn, isClient)

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

func del(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
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
		return "", errors.Wrapf(err, "failed to get network namespace handle for %s", u.Path)
	}
	defer func() { _ = targetNetNS.Close() }()

	// VPP uses "abstract:" notation to create an abstract socket. But once created on Unix, it has "@" prefix.
	// According to the VPP API we have to replace "@" with "abstract:"
	vppSocketFilename := strings.ReplaceAll(mechanism.GetSocketFilename(), "@", "abstract:")
	if !targetNetNS.Equal(netNS) {
		return fmt.Sprintf("%s,netns_name=%s", vppSocketFilename, u.Path), nil
	}
	return vppSocketFilename, nil
}
