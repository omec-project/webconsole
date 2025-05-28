// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

type ServiceConfiguration struct {
	TLS struct {
		Enabled bool   `yaml:"enabled"`
		Key     string `yaml:"key"`
		Pem     string `yaml:"pem"`
	} `yaml:"tls"`
}

// TODO: implement the config models in the next PRs

type AccessMobilityConfig struct{}

type PlmnConfig struct{}

type PlmnSnssaiConfig struct{}

type SessionManagementConfig struct{}

type PolicyControlConfig struct{}
