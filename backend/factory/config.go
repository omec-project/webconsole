// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

/*

 * WebUi Configuration Factory

 */

package factory

import (
	"github.com/omec-project/util/logger"
)

type Config struct {
	Info          *Info          `yaml:"info"`
	Configuration *Configuration `yaml:"configuration"`
	Logger        *logger.Logger `yaml:"logger"`
}

type Info struct {
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
	HttpVersion int    `yaml:"http-version,omitempty"`
}

type Configuration struct {
	Mongodb                 *Mongodb  `yaml:"mongodb"`
	WebuiTLS                *TLS      `yaml:"webui-tls"`
	NfConfigTLS             *TLS      `yaml:"nfconfig-tls"`
	RocEnd                  *RocEndpt `yaml:"managedByConfigPod,omitempty"` // fetch config during bootup
	SdfComp                 bool      `yaml:"spec-compliant-sdf"`
	EnableAuthentication    bool      `yaml:"enableAuthentication,omitempty"`
	SendPebbleNotifications bool      `yaml:"send-pebble-notifications,omitempty"`
	CfgPort                 int       `yaml:"cfgport,omitempty"`
	SSM                     *SSM      `yaml:"ssm,omitempty"`
	Vault                   *Vault    `yaml:"vault,omitempty"`
}

type SSM struct {
	SsmUri          string    `yaml:"ssm-uri,omitempty"`
	AllowSsm        bool      `yaml:"allow-ssm,omitempty"`
	TLS_Insecure    bool      `yaml:"tls-insecure,omitempty"`
	SsmSync         *SsmSync  `yaml:"ssm-synchronize,omitempty"`
	MTls            *TLS2     `yaml:"m-tls,omitempty"`
	Login           *SSMLogin `yaml:"login,omitempty"` // use this config only for development purposes use environment variables in production
	IsEncryptAESCBC bool      `yaml:"is-encrypt-aes-cbc,omitempty"`
	IsEncryptAESGCM bool      `yaml:"is-encrypt-aes-gcm,omitempty"`
}

type Vault struct {
	VaultUri       string   `yaml:"vault-uri,omitempty"`
	AllowVault     bool     `yaml:"allow-vault,omitempty"`
	Token          string   `yaml:"token,omitempty"`
	MountApp       string   `yaml:"mount-app,omitempty"`
	TLS_Insecure   bool     `yaml:"tls-insecure,omitempty"`
	MTls           *TLS2    `yaml:"m-tls,omitempty"`
	CertRole       string   `yaml:"cert-role,omitempty"`
	K8sRole        string   `yaml:"k8s-role,omitempty"`
	K8sJWTPath     string   `yaml:"k8s-jwt-path,omitempty"`
	RoleID         string   `yaml:"role-id,omitempty"`
	SecretID       string   `yaml:"secret-id,omitempty"`
	ConcurrencyOps int16    `yaml:"concurrency-ops,omitempty"`
	SsmSync        *SsmSync `yaml:"ssm-synchronize,omitempty"`

	// Auth mount paths for custom Vault configurations
	AppRoleMountPath string `yaml:"approle-mount-path,omitempty"` // e.g., "approle" (default) or custom mount
	K8sMountPath     string `yaml:"k8s-mount-path,omitempty"`     // e.g., "kubernetes" (default) or custom mount
	CertMountPath    string `yaml:"cert-mount-path,omitempty"`    // e.g., "cert" (default) or custom mount

	// Paths and formats for Vault KV and Transit
	KeyKVPath              string `yaml:"key-kv-path,omitempty"`               // e.g., "secret/data/k4keys"
	KeyKVMetadataPath      string `yaml:"key-kv-metadata-path,omitempty"`      // e.g., "secret/metadata/k4keys"
	TransitKeysListPath    string `yaml:"transit-keys-list-path,omitempty"`    // e.g., "transit/keys"
	TransitKeyCreateFmt    string `yaml:"transit-key-create-fmt,omitempty"`    // e.g., "transit/keys/%s"
	TransitKeyRotateFmt    string `yaml:"transit-key-rotate-fmt,omitempty"`    // e.g., "transit/keys/%s/rotate"
	TransitKeyRewrapFmt    string `yaml:"transit-key-rewrap-fmt,omitempty"`    // e.g., "transit/rewrap/%s"
	TransitKeysEncryptPath string `yaml:"transit-keys-encrypt-path,omitempty"` // e.g., "transit/encrypt"
}

type TLS struct {
	PEM string `yaml:"pem,omitempty"`
	Key string `yaml:"key,omitempty"`
	Ca  string `yaml:"ca,omitempty"`
}

type TLS2 struct {
	Crt string `yaml:"crt,omitempty"`
	Key string `yaml:"key,omitempty"`
	Ca  string `yaml:"ca,omitempty"`
}

type SSMLogin struct {
	ServiceId string `yaml:"service-id,omitempty"`
	Password  string `yaml:"password,omitempty"`
}

type SsmSync struct {
	Enable           bool `yaml:"enable,omitempty"`
	IntervalMinute   int  `yaml:"interval-minute,omitempty"`
	MaxKeysCreate    int  `yaml:"max-keys-create,omitempty"`
	DeleteMissing    bool `yaml:"delete-missing,omitempty"`
	MaxSyncKeys      int  `yaml:"max-sync-keys,omitempty"`
	MaxSyncUsers     int  `yaml:"max-sync-users,omitempty"`
	MaxSyncRotations int  `yaml:"max-sync-rotations,omitempty"`
}

type Mongodb struct {
	Name           string `yaml:"name,omitempty"`
	Url            string `yaml:"url,omitempty"`
	DefaultConns   int    `yaml:"defaultConns,omitempty"`
	AuthKeysDbName string `yaml:"authKeysDbName"`
	AuthUrl        string `yaml:"authUrl"`
	AuthConns      int    `yaml:"authConns"`
	WebuiDBName    string `yaml:"webuiDbName,omitempty"`
	WebuiDBUrl     string `yaml:"webuiDbUrl,omitempty"`
	WebuiDbConns   int    `yaml:"webuiDbConns"`
	CheckReplica   bool   `yaml:"checkReplica,omitempty"`
	ConcurrencyOps int    `yaml:"concurrency-ops,omitempty"`
}

type RocEndpt struct {
	SyncUrl string `yaml:"syncUrl,omitempty"`
	Enabled bool   `yaml:"enabled,omitempty"`
}

type LteEndpt struct {
	NodeType       string `yaml:"type,omitempty"`
	ConfigPushUrl  string `yaml:"configPushUrl,omitempty"`
	ConfigCheckUrl string `yaml:"configCheckUrl,omitempty"` // only for 4G components
}
