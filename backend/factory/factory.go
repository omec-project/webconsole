// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * WebUI Configuration Factory
 */

package factory

import (
	"fmt"
	"os"

	openapiLogger "github.com/omec-project/openapi/v2/logger"
	utilLogger "github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

var WebUIConfig *Config

func init() {
	WebUIConfig = &Config{Configuration: &Configuration{CfgPort: 5000}}
}

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	content, err := os.ReadFile(f)
	if err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	}
	if err = yaml.Unmarshal(content, WebUIConfig); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	}
	if WebUIConfig.Configuration.WebuiTLS != nil {
		if WebUIConfig.Configuration.WebuiTLS.Key == "" ||
			WebUIConfig.Configuration.WebuiTLS.PEM == "" {
			return fmt.Errorf("[WebUI Configuration] TLS Key and PEM must be set")
		}
	}
	if WebUIConfig.Configuration.NfConfigTLS != nil {
		if WebUIConfig.Configuration.NfConfigTLS.Key == "" ||
			WebUIConfig.Configuration.NfConfigTLS.PEM == "" {
			return fmt.Errorf("[NFConfig Configuration] TLS Key and PEM must be set")
		}
	}
	if WebUIConfig.Configuration.Mongodb.AuthUrl == "" {
		authUrl := WebUIConfig.Configuration.Mongodb.Url
		WebUIConfig.Configuration.Mongodb.AuthUrl = authUrl
	}
	if WebUIConfig.Configuration.Mongodb.AuthKeysDbName == "" {
		WebUIConfig.Configuration.Mongodb.AuthKeysDbName = "authentication"
	}

	if WebUIConfig.Configuration.EnableAuthentication {
		if WebUIConfig.Configuration.Mongodb.WebuiDBName == "" ||
			WebUIConfig.Configuration.Mongodb.WebuiDBUrl == "" {
			return fmt.Errorf("[Configuration] if EnableAuthentication is set, WebuiDB must be set")
		}
	}

	if WebUIConfig.Configuration.RocEnd != nil {
		if WebUIConfig.Configuration.RocEnd.Enabled && WebUIConfig.Configuration.RocEnd.SyncUrl == "" {
			return fmt.Errorf("[Configuration] if RocEnd enabled, SyncUrl must be set")
		}
	}

	return nil
}

func SetLogLevelsFromConfig(cfg *Config) {
	cfgLogger := cfg.Logger
	if cfgLogger == nil {
		logger.InitLog.Warnln("webconsole config without log level setting")
		return
	}

	utilLogger.ApplyLogSetting("WEBUI", cfgLogger.WEBUI, logger.InitLog, logger.SetLogLevel)
	utilLogger.ApplyLogSetting("OpenApi", cfgLogger.OpenApi, openapiLogger.OpenapiLog, openapiLogger.SetLogLevel)
	utilLogger.ApplyLogSetting("Util", cfgLogger.Util, utilLogger.UtilLog, utilLogger.SetLogLevel)
}

func GetCliFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "cfg",
			Usage: "Path to configuration file",
		},
	}
}
