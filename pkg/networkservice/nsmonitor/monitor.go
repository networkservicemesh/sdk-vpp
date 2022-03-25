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

package nsmonitor

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
)

const (
	monitorInterval = 250 * time.Millisecond

	monitorResultAdded             = "added to monitoring"
	monitorResultAlreadyMonitored  = "already monitored"
	monitorResultUnsupportedScheme = "unsupported scheme"
)

// getProcName returns process name by its pid
func getProcName(pid uint64) (string, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/stat", pid))
	if err != nil {
		return "", err
	}
	data := string(bytes)
	start := strings.IndexRune(data, '(') + 1
	end := strings.IndexRune(data[start:], ')')
	return data[start : start+end], nil
}

// getInode returns Inode for file
func getInode(file string) (uint64, error) {
	fileinfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return stat.Ino, nil
}

type netNSInfo struct {
	pid   uint64
	inode uint64
}

// getAllNetNS returns all network namespace inodes and associated process pids
func getAllNetNS() ([]netNSInfo, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, errors.Wrap(err, "can't read /proc directory")
	}
	var inodes []netNSInfo
	for _, f := range files {
		pid, err := strconv.ParseUint(f.Name(), 10, 64)
		if err != nil {
			continue
		}
		inode, err := getInode(fmt.Sprintf("/proc/%v/ns/net", pid))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return nil, err
			}
		}
		inodes = append(inodes, netNSInfo{
			pid:   pid,
			inode: inode,
		})
	}
	return inodes, nil
}

type monitorItem struct {
	ctx   context.Context
	pause string
}

type netNSMonitor struct {
	mutex sync.Mutex
	nses  map[uint64]monitorItem
	on    bool
}

func newMonitor() *netNSMonitor {
	return &netNSMonitor{
		nses: map[uint64]monitorItem{},
	}
}

func (m *netNSMonitor) AddNSInode(ctx context.Context, nsInodeURL string) (string, error) {
	inodeURL, err := url.Parse(nsInodeURL)
	if err != nil {
		return "", errors.Wrap(err, "invalid url")
	}

	if inodeURL.Scheme != "inode" {
		// We also receive smth like file:///proc/2608554/fd/37
		// it's not an error
		return monitorResultUnsupportedScheme, nil
	}

	pathParts := strings.Split(inodeURL.Path, "/")
	inode, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "invalid inode path")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.nses[inode]; ok {
		return monitorResultAlreadyMonitored, nil
	}

	nses, err := getAllNetNS()
	if err != nil {
		return "", errors.Wrap(err, "unable to get all netns")
	}

	var pausePid uint64 = 0
	for _, ns := range nses {
		if ns.inode == inode {
			proc, err := getProcName(ns.pid)
			if err != nil {
				return "", errors.Wrap(err, "unable to get proc name")
			}
			if proc == "pause" {
				pausePid = ns.pid
			}
		}
	}
	if pausePid == 0 {
		return "", errors.New("pause container not found")
	}

	m.nses[inode] = monitorItem{
		ctx:   ctx,
		pause: fmt.Sprintf("/proc/%v", pausePid),
	}
	if !m.on {
		m.on = true
		go m.monitor()
	}
	return monitorResultAdded, nil
}

func (m *netNSMonitor) monitor() {
	for {
		<-time.After(monitorInterval)

		if !m.checkAllNS() {
			return
		}
	}
}

func (m *netNSMonitor) checkAllNS() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var toDelete []uint64
	for inode, item := range m.nses {
		if _, err := os.Stat(item.pause); err != nil {
			println("netNsMonitor", fmt.Sprintf("%v died", inode))
			toDelete = append(toDelete, inode)
		}
	}

	if len(toDelete) > 0 {
		for _, inode := range toDelete {
			begin.FromContext(m.nses[inode].ctx).Close()
			delete(m.nses, inode)
		}

		if len(m.nses) == 0 {
			m.on = false
			return false
		}
	}

	return true
}
