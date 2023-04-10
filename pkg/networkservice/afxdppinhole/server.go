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
	"os"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type afxdppinholeServer struct {
	elfPath  string
	bpfFSDir string
}

// NewServer - returns a new client that updates the UDP ports map for NSM interfaces
func NewServer(options ...Option) networkservice.NetworkServiceServer {
	opts := &afxdpOptions{
		elfPath:  defaultElfPath,
		bpfFSDir: defaultBpfFsDir,
	}
	for _, opt := range options {
		opt(opts)
	}
	return &afxdppinholeServer{
		elfPath:  opts.elfPath,
		bpfFSDir: opts.bpfFSDir,
	}
}

func (s *afxdppinholeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Check if we use af_xdp
	if _, err := os.Stat(path.Join(s.bpfFSDir, defaultXDPPinholeMapName)); errors.Is(err, os.ErrNotExist) {
		return next.Server(ctx).Request(ctx, request)
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Get UDP ports from the mechanism and context
	keys := []uint16{
		fromMechanism(conn.GetMechanism(), metadata.IsClient(s)),
		fromContext(ctx, metadata.IsClient(s)),
	}

	if err = updateXdpPinhole(keys, s.elfPath, s.bpfFSDir); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()
		if _, closeErr := s.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}
		return nil, err
	}

	return conn, nil
}

func (s *afxdppinholeServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
