// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/*

 * WebUi Configuration Factory

 */

package factory

import (
	"github.com/free5gc/logger_util"
)

type Config struct {
	Info          *Info               `yaml:"info"`
	Configuration *Configuration      `yaml:"configuration"`
	Logger        *logger_util.Logger `yaml:"logger"`
}

type Info struct {
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Configuration struct {
	Mode5G    bool        `yaml:"mode5G,omitempty"`
	WebServer *WebServer  `yaml:"WebServer,omitempty"`
	Mongodb   *Mongodb    `yaml:"mongodb"`
	RocEnd    *RocEndpt   `yaml:"managedByConfigPod,omitempty"` // fetch config during bootup
	LteEnd    []*LteEndpt `yaml:"endpoints,omitempty"`          //LTE endpoints are configured and not auto-detected
}

type WebServer struct {
	Scheme string `yaml:"scheme"`
	IP     string `yaml:"ipv4Address"`
	PORT   string `yaml:"port"`
}

type Mongodb struct {
	Name string `yaml:"name,omitempty"`
	Url  string `yaml:"url,omitempty"`
}

type RocEndpt struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	SyncUrl string `yaml:"syncUrl,omitempty"`
}

type LteEndpt struct {
	NodeType string `yaml:"type,omitempty"`
	ConfigPushUrl   string `yaml:"configPushUrl,omitempty"`
	ConfigCheckUrl string `yaml:"configCheckUrl,omitempty"`  // only for 4G components 
}
