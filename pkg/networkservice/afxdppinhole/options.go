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

const (
	defaultElfPath           = "/bin/afxdp.o"
	defaultBpfFsDir          = "/sys/fs/bpf"
	defaultXDPPinholeMapName = "nsm_xdp_pinhole"
)

type afxdpOptions struct {
	// path to af_xdp object file
	elfPath string
	// BPF filesystem directory
	bpfFSDir string
}

// Option is an option pattern for afxdppinhole chain elements
type Option func(o *afxdpOptions)

// WithElfPath sets the path of the ELF object file
func WithElfPath(p string) Option {
	return func(o *afxdpOptions) {
		o.elfPath = p
	}
}

// WithBpfFsDir sets the dir of the BPF file system
func WithBpfFsDir(p string) Option {
	return func(o *afxdpOptions) {
		o.bpfFSDir = p
	}
}
