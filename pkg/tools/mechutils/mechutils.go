// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

// Package mechutils provides utilities for conververtin kernel.Mechanism to various things
package mechutils

import (
	"fmt"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
)

// ToNSFilename - mechanism to NetNS filename
func ToNSFilename(mechanism *kernel.Mechanism) (string, error) {
	u, err := url.Parse(mechanism.GetNetNSURL())
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse url %s", mechanism.GetNetNSURL())
	}
	if u.Scheme != kernel.NetNSURLScheme {
		return "", errors.Errorf("NetNSURL Scheme required to be %q actual %q", kernel.NetNSURLScheme, u.Scheme)
	}
	if u.Path == "" {
		return "", errors.Errorf("NetNSURL.Path canot be empty %q", u.Path)
	}
	return u.Path, nil
}

// ToAlias - create interface alias/tag from conn for client or server side for forwarder.
//
//	Note: Don't use this in a non-forwarder context
func ToAlias(conn *networkservice.Connection, isClient bool) string {
	// Naming is tricky.  We want to name based on either the next or prev connection id depending on whether we
	// are on the client or server side.  Since this chain element is designed for use in a Forwarder,
	// if we are on the client side, we want to name based on the connection id from the NSE that is Next
	// if we are not the client, we want to name for the connection of of the client addressing us, which is Prev
	namingConn := conn.Clone()
	namingConn.Id = namingConn.GetPrevPathSegment().GetId()
	alias := fmt.Sprintf("server-%s", namingConn.GetId())
	if isClient {
		namingConn.Id = namingConn.GetNextPathSegment().GetId()
		alias = fmt.Sprintf("client-%s", namingConn.GetId())
	}
	return alias
}
