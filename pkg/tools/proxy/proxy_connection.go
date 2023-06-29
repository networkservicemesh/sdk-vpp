// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package proxy

import (
	"net"
	"syscall"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	maxFDCount = 1
	bufferSize = 128
)

type proxyConnection struct {
	in  net.Conn
	out net.Conn
}

func newProxyConnection(in, out net.Conn) (*proxyConnection, error) {
	p := &proxyConnection{
		in:  in,
		out: out,
	}
	if err := p.copy(in, out); err != nil {
		return nil, err
	}
	if err := p.copy(out, in); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *proxyConnection) Close() error {
	inErr := p.in.Close()
	outErr := p.out.Close()
	if inErr != nil {
		return multierror.Append(inErr, outErr)
	}
	return outErr
}

func (p *proxyConnection) copy(dst, src net.Conn) error {
	b := make([]byte, bufferSize)
	unixsrc, unixSrcOK := src.(interface {
		ReadMsgUnix(b, oob []byte) (n, oobn, flags int, addr *net.UnixAddr, err error)
	})
	if !unixSrcOK {
		return errors.Errorf("%s does not implement ReadMsgUnix", src.LocalAddr())
	}

	unixdst, unixdstOK := dst.(interface {
		WriteMsgUnix(b, oob []byte, addr *net.UnixAddr) (n, oobn int, err error)
	})

	if !unixdstOK {
		return errors.Errorf("%s does not implement ReadMsgUnix", dst.LocalAddr())
	}

	go func() {
		oob := make([]byte, syscall.CmsgSpace(4*maxFDCount))
		for {
			var writeN, writeoob int
			readn, readoobn, _, _, err := unixsrc.ReadMsgUnix(b, oob)
			if err != nil {
				return
			}
			for writeN < readn {
				writeN, writeoob, err = unixdst.WriteMsgUnix(b[writeN:readn], oob[writeoob:readoobn], nil)
				if err != nil {
					return
				}
			}
		}
	}()
	return nil
}
