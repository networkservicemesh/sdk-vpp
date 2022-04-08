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

// Package tagtool provides some utilities for converting NSM interface tag
package tagtool

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	clientIdentifier = "c"
	serverIdentifier = "s"
	delimiter        = "_"
)

// Tag - represent an NSM tag
type Tag struct {
	TagPrefix string
	ConnID    string
	IsClient  bool
}

// ConvertToString - converts Tag to the string format: { forwarder-name }_{ identifier }_{ connID }
func ConvertToString(t *Tag) string {
	isClientStr := clientIdentifier
	if !t.IsClient {
		isClientStr = serverIdentifier
	}
	return t.TagPrefix + delimiter + isClientStr + delimiter + t.ConnID
}

// ConvertFromString - converts from string to Tag
func ConvertFromString(tag string) (t *Tag, err error) {
	if tag == "" {
		return nil, errors.New("tag is empty")
	}
	substrs := strings.Split(tag, delimiter)
	if len(substrs) != 3 {
		return nil, errors.New("tag part count mismatch")
	}

	t = &Tag{
		TagPrefix: substrs[0],
		ConnID:    substrs[2],
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
