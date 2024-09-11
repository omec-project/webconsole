// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configmodels

type User struct {
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	Permissions int    `json:"permissions"`
}
