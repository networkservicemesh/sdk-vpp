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
	"os"
	"path/filepath"
	"time"

	"git.fd.io/govpp.git"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"gopkg.in/fsnotify.v1"
)

type connection struct {
	*core.Connection
	ready chan struct{}
	err   error
}

// DialContext - Dials vpp and returns a Connection
// DialContext is 'lazy' meaning that if there is no socket yet at filename, we will continue to try
// until there is one or the ctx is canceled.
func DialContext(ctx context.Context, filename string) Connection {
	c := &connection{
		ready: make(chan struct{}),
	}
	go c.connect(ctx, filename)
	return c
}

func (c *connection) connect(ctx context.Context, filename string) {
	defer close(c.ready)
	now := time.Now()
	c.err = waitForSocket(ctx, filename)
	if c.err != nil {
		log.FromContext(ctx).Debugf("%s was not created after %s due to %+v", filename, time.Since(now), c.err)
		return
	}
	log.FromContext(ctx).Debugf("%s was created after %s", filename, time.Since(now))
	now = time.Now()
	attempts := 1
	for {
		select {
		case <-ctx.Done():
			c.err = errors.WithStack(ctx.Err())
			log.FromContext(ctx).Debugf("unable to connect to %s after %s due to %+v", filename, time.Since(now), c.err)
			return
		default:
			c.Connection, c.err = govpp.Connect(filename)
			if c.err == nil {
				log.FromContext(ctx).Debugf("successfully connected to %s after %s and %d attempts", filename, time.Since(now), attempts)
				return
			}
			attempts++
			<-time.After(time.Millisecond)
		}
	}
}

func (c *connection) NewStream(ctx context.Context, options ...api.StreamOption) (api.Stream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ready:
		if c.err != nil {
			return nil, c.err
		}
	}
	return c.Connection.NewStream(ctx, options...)
}

func (c *connection) Invoke(ctx context.Context, req, reply api.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ready:
		if c.err != nil {
			return c.err
		}
	}
	return c.Connection.Invoke(ctx, req, reply)
}

func (c *connection) NewAPIChannel() (api.Channel, error) {
	<-c.ready
	if c.err != nil {
		return nil, c.err
	}
	return c.Connection.NewAPIChannel()
}

func (c *connection) NewAPIChannelBuffered(reqChanBufSize, replyChanBufSize int) (api.Channel, error) {
	<-c.ready
	if c.err != nil {
		return nil, c.err
	}
	return c.Connection.NewAPIChannelBuffered(reqChanBufSize, replyChanBufSize)
}

var _ Connection = &connection{}

func waitForSocket(ctx context.Context, filename string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() { _ = watcher.Close() }()
	if err = watcher.Add(filepath.Dir(filename)); err != nil {
		return errors.WithStack(err)
	}

	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				if event.Name == filename && event.Op == fsnotify.Create {
					return nil
				}
				// watch for errors
			case err = <-watcher.Errors:
				return errors.WithStack(err)
			case <-ctx.Done():
				return errors.WithStack(ctx.Err())
			}
		}
	}
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
