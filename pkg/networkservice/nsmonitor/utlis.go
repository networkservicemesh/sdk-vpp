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

package nsmonitor

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
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

// getInode returns inode for file
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

// parseInode converts inode URL string into inode handle
func parseInode(nsInodeURL string) (uint64, error) {
	inodeURL, err := url.Parse(nsInodeURL)
	if err != nil {
		return 0, errors.Wrap(err, "invalid url")
	}

	if inodeURL.Scheme != "inode" {
		return 0, errors.New("unsupported scheme")
	}

	pathParts := strings.Split(inodeURL.Path, "/")
	inode, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid inode path")
	}

	return inode, nil
}

// findPauseProc finds a pause process for specified netns inode
func findPauseProc(inode uint64) (netNSInfo, error) {
	nses, err := getAllNetNS()
	if err != nil {
		return netNSInfo{}, errors.Wrap(err, "unable to get all netns")
	}

	for _, ns := range nses {
		if ns.inode == inode {
			proc, err := getProcName(ns.pid)
			if err != nil {
				return netNSInfo{}, errors.Wrap(err, "unable to get proc name")
			}
			if proc == "pause" {
				return ns, nil
			}
		}
	}

	return netNSInfo{}, errors.New("pause container not found")
}

// getMonitoringTarget returns netns inode handle and its pause container process path
func getTarget(nsInodeURL string) (inode uint64, pause string, err error) {
	inode, err = parseInode(nsInodeURL)
	if err != nil {
		return 0, "", err
	}

	nsInfo, err := findPauseProc(inode)
	if err != nil {
		return 0, "", err
	}

	return inode, fmt.Sprintf("/proc/%v", nsInfo.pid), nil
}
