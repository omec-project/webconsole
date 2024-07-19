// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package configmodels

type Gnb struct {
	GnbName string `json:"gnbName"`
	Tac     string `json:"tac"`
}

type Upf struct {
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
}