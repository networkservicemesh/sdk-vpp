// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

func resolveProcByInodeURL(inodeURL string) (string, error) {
	inode, err := parseInode(inodeURL)
	if err != nil {
		return "", err
	}
	candidates, err := ioutil.ReadDir("/proc")
	if err != nil {
		return "", errors.Wrap(err, "failed to read directory /proc")
	}
	for _, f := range candidates {
		pid, err := strconv.ParseUint(f.Name(), 10, 64)
		if err != nil {
			continue
		}
		candidateInode, err := inodeFromString(fmt.Sprintf("/proc/%v/ns/net", pid))
		if err != nil {
			continue
		}
		if candidateInode == inode {
			procName, err := getProcName(pid)
			if err != nil || procName != "pause" {
				continue
			}
			return fmt.Sprintf("/proc/%v", pid), nil
		}
	}
	return "", errors.Errorf("inode %v is not found in /proc", inode)
}

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

func inodeFromString(file string) (uint64, error) {
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

// parseInode converts inode URL string into inode handle
func parseInode(nsInodeURL string) (uint64, error) {
	inodeURL, err := url.Parse(nsInodeURL)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid url %s", nsInodeURL)
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
