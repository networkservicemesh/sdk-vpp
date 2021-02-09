// Copyright (c) 2020 Cisco and/or its affiliates.
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

package memif

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"git.fd.io/govpp.git/api"
	interfaces "github.com/edwarnicke/govpp/binapi/interface"
	"github.com/edwarnicke/govpp/binapi/interface_types"
	"github.com/edwarnicke/govpp/binapi/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	memifMech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/tools/ifindex"
)

var lastSocketID uint32

func createMemifSocket(ctx context.Context, mechanism *memifMech.Mechanism, vppConn api.Connection, isClient bool) (socketID uint32, err error) {
	// Extract the socket filename
	u, err := url.Parse(mechanism.GetSocketFileURL())
	if err != nil {
		return 0, errors.Wrapf(err, "not a valid url %q", mechanism.GetSocketFileURL())
	}
	if u.Scheme != memifMech.SocketFileScheme {
		return 0, errors.Errorf("socket file url must have scheme %q, actual %q", memifMech.SocketFileScheme, u.Scheme)
	}

	// Create the socketID
	socketID = atomic.AddUint32(&lastSocketID, 1) // TODO - work out a solution that works long term
	now := time.Now()
	memifSocketAddDel := &memif.MemifSocketFilenameAddDel{
		IsAdd:          true,
		SocketID:       socketID,
		SocketFilename: u.Path,
	}
	if _, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDel(ctx, memifSocketAddDel); err != nil {
		return 0, errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("SocketID", memifSocketAddDel.SocketID).
		WithField("SocketFilename", memifSocketAddDel.SocketFilename).
		WithField("IsAdd", memifSocketAddDel.IsAdd).
		WithField("duration", time.Since(now)).
		WithField("vppapi", "MemifSocketFilenameAddDel").Debug("completed")
	store(ctx, isClient, memifSocketAddDel)
	return socketID, nil
}

func deleteMemifSocket(ctx context.Context, vppConn api.Connection, isClient bool) error {
	memifSocketAddDel, ok := load(ctx, isClient)
	if !ok {
		return nil
	}
	memifSocketAddDel.IsAdd = false
	now := time.Now()
	if _, err := memif.NewServiceClient(vppConn).MemifSocketFilenameAddDel(ctx, memifSocketAddDel); err != nil {
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

func createMemif(ctx context.Context, vppConn api.Connection, socketID uint32, isClient bool) error {
	role := memif.MEMIF_ROLE_API_MASTER
	if isClient {
		role = memif.MEMIF_ROLE_API_SLAVE
	}
	now := time.Now()
	memifCreate := &memif.MemifCreate{
		Role:     role,
		SocketID: socketID,
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
	swIfIndex, ok := ifindex.Load(ctx, isClient)
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

func create(ctx context.Context, conn *networkservice.Connection, vppConn api.Connection, isClient bool) error {
	if mechanism := memifMech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// Direct memif if applicable
		if memifSocketAddDel, ok := load(ctx, true); ok && !isClient {
			_, ok := ifindex.Load(ctx, !isClient)
			if ok {
				if err := del(ctx, conn, vppConn, !isClient); err != nil {
					return err
				}
				mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: memifSocketAddDel.SocketFilename}).String())
				dElete(ctx, !isClient)
				ifindex.Delete(ctx, !isClient)
				return nil
			}
		}
		// This connection has already been created
		if _, ok := ifindex.Load(ctx, isClient); ok {
			return nil
		}
		if !isClient {
			if err := os.MkdirAll(filepath.Dir(socketFile(conn)), 0700); err != nil {
				return errors.Wrapf(err, "failed to create memif socket directory %s", socketFile(conn))
			}
			mechanism.SetSocketFileURL((&url.URL{Scheme: memifMech.SocketFileScheme, Path: socketFile(conn)}).String())
		}
		socketID, err := createMemifSocket(ctx, mechanism, vppConn, isClient)
		if err != nil {
			return err
		}
		if err := createMemif(ctx, vppConn, socketID, isClient); err != nil {
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
		if !isClient {
			if err := os.RemoveAll(filepath.Dir(socketFile(conn))); err != nil {
				return errors.Wrapf(err, "failed to delete %s", filepath.Dir(socketFile(conn)))
			}
		}
	}
	return nil
}

func socketFile(conn *networkservice.Connection) string {
	return filepath.Join(os.TempDir(), "memif", conn.GetId(), "memif.socket")
}
