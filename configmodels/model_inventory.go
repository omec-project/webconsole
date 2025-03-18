// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configmodels

const (
	GnbDataColl = "webconsoleData.snapshots.gnbData"
	UpfDataColl = "webconsoleData.snapshots.upfData"
)

type Gnb struct {
	Name string `json:"name"`
	Tac  int32  `json:"tac,omitempty"`
}

type PostGnbRequest struct {
	Name string `json:"name"`
	Tac  int32  `json:"tac,omitempty"`
}

type PutGnbRequest struct {
	Tac int32 `json:"tac"`
}

type Upf struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
}

type PostUpfRequest struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
}

type PutUpfRequest struct {
	Port string `json:"port"`
}
