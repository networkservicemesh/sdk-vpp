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

package dumptool

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	clientIdentifier = "c"
	serverIdentifier = "s"
	delimiter        = "_"

	// DevTypeTap -
	DevTypeTap = "virtio"
	// DevTypeMemif -
	DevTypeMemif = "memif"
	// DevTypeMemif -
	DevTypeVxlan = "VXLAN"
)

// TagStruct -
type TagStruct struct {
	PodName  string
	ConnID   string
	IsClient bool
}

// ConvertToTag - format: forwarder-xxx_c_0000-0000-0000-0000
func ConvertToTag(t *TagStruct) string {
	isClientStr := clientIdentifier
	if !t.IsClient {
		isClientStr = serverIdentifier
	}
	return t.PodName + delimiter + isClientStr + delimiter + t.ConnID
}

// ConvertFromTag - format: forwarder-xxx_c_0000-0000-0000-0000
func ConvertFromTag(tag string) (t *TagStruct, err error) {
	if tag == "" {
		return nil, errors.New("tag is empty")
	}
	substrs := strings.Split(tag, delimiter)
	if len(substrs) != 3 {
		return nil, errors.New("tag part count mismatch")
	}

	t = &TagStruct{
		PodName: substrs[0],
		ConnID:  substrs[2],
	}
	switch substrs[1] {
	case clientIdentifier:
		t.IsClient = true
	case serverIdentifier:
		t.IsClient = false
	default:
		return nil, errors.New("identifier mismatch")
	}
	return t, nil
}
