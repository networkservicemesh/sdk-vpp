// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

// Package ethtool provides some utilities for disabling checksum offload using ethtool
package ethtool

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	siocEthtool = 0x8946 // linux/sockios.h

	ethtoolSRxCsum = 0x00000015 // linux/ethtoolDisableOffload.h

	ethtoolSTxCsum = 0x00000017 // linux/ethtoolDisableOffload.h

	maxIfNameSize = 16 // linux/if.h
)

// linux/if.h 'struct ifreq'
type ifreq struct {
	Name [maxIfNameSize]byte
	Data uintptr
}

// linux/ethtoolDisableOffload.h 'struct ethtool_value'
type ethtoolValue struct {
	Cmd  uint32
	Data uint32
}

// ethtoolDisableOffload executes Linux ethtoolDisableOffload syscall.
func ethtoolDisableOffload(iface string, cmd uint32) (retval uint32, err error) {
	if len(iface)+1 > maxIfNameSize {
		return 0, errors.Errorf("interface name is too long")
	}
	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return 0, err
	}
	defer func() { _ = syscall.Close(socket) }()

	// prepare ethtoolDisableOffload request
	value := ethtoolValue{cmd, 0}
	request := ifreq{Data: uintptr(unsafe.Pointer(&value))} // #nosec
	copy(request.Name[:], iface)

	// ioctl system call
	_, _, errno := syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(socket), uintptr(siocEthtool),
		uintptr(unsafe.Pointer(&request))) // #nosec
	if errno != 0 {
		return 0, errno
	}
	return value.Data, nil
}

// DisableVethChkSumOffload - disables ChkSumOffload for Veth
func DisableVethChkSumOffload(veth *netlink.Veth) error {
	retval, err := ethtoolDisableOffload(veth.LinkAttrs.Name, ethtoolSTxCsum)
	if err != nil {
		return errors.Wrapf(err, "with retval %d", retval)
	}
	retval, err = ethtoolDisableOffload(veth.LinkAttrs.Name, ethtoolSRxCsum)
	if err != nil {
		return errors.Wrapf(err, "with retval %d", retval)
	}
	retval, err = ethtoolDisableOffload(veth.PeerName, ethtoolSTxCsum)
	if err != nil {
		return errors.Wrapf(err, "with retval %d", retval)
	}
	retval, err = ethtoolDisableOffload(veth.PeerName, ethtoolSRxCsum)
	if err != nil {
		return errors.Wrapf(err, "with retval %d", retval)
	}
	return nil
}
