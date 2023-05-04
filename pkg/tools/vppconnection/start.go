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

package vppconnection

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"git.fd.io/govpp.git/api"
	"github.com/edwarnicke/exechelper"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/sirupsen/logrus"
)

// Connection Combination of api.Connection and api.ChannelProvider
type Connection interface {
	api.Connection
	api.ChannelProvider
}

// StartAndDialContext - starts vpp
// Stdout and Stderr for vpp are set to be logrus.Entry.Writer().
func StartAndDialContext(ctx context.Context, opts ...Option) (conn Connection, errCh <-chan error) {
	o := &option{
		rootDir:   DefaultRootDir,
		vppConfig: vppConfContents,
	}
	for _, opt := range opts {
		opt(o)
	}

	if err := writeDefaultConfigFiles(ctx, o); err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		return nil, errCh
	}
	logWriter := logrus.StandardLogger().WithField("cmd", "vpp").Writer()
	vppErrCh := exechelper.Start("vpp -c "+filepath.Join(o.rootDir, vppConfFilename),
		exechelper.WithContext(ctx),
		exechelper.WithStdout(logWriter),
		exechelper.WithStderr(logWriter),
	)
	select {
	case err := <-vppErrCh:
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		return nil, errCh
	default:
	}

	return DialContext(ctx, filepath.Join(o.rootDir, "/var/run/vpp/api.sock")), vppErrCh
}

func writeDefaultConfigFiles(ctx context.Context, o *option) error {
	configFiles := map[string]string{
		vppConfFilename: fmt.Sprintf(o.vppConfig, o.rootDir),
	}
	for filename, contents := range configFiles {
		filename = filepath.Join(o.rootDir, filename)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			log.FromContext(ctx).Infof("Configuration file: %q not found, using defaults", filename)
			if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
				return err
			}
			if err := ioutil.WriteFile(filename, []byte(contents), 0600); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(o.rootDir, "/var/run/vpp"), 0700); os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(o.rootDir, "/var/log/vpp"), 0700); os.IsNotExist(err) {
		return err
	}
	return nil
}
