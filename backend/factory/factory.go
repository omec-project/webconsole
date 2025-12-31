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

	utilLogger "github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

var WebUIConfig *Config

func init() {
	WebUIConfig = &Config{Configuration: &Configuration{CfgPort: 5000}}
}

func GetConfig() *Config {
	return WebUIConfig
}

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	content, err := os.ReadFile(f)
	if err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	}

	// expande ${VAR} y $VAR desde el entorno
	expanded := []byte(os.ExpandEnv(string(content)))

	if yamlErr := yaml.Unmarshal(expanded, WebUIConfig); yamlErr != nil {
		return fmt.Errorf("[Configuration] %+v", yamlErr)
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

	if WebUIConfig.Configuration.Mongodb.ConcurrencyOps == 0 {
		WebUIConfig.Configuration.Mongodb.ConcurrencyOps = 10
	}

	logger.AppLog.Infof("The ssm config is: %s", WebUIConfig.Configuration.SSM)
	if WebUIConfig.Configuration.SSM == nil {
		logger.AppLog.Info("The ssm config is empty")
		WebUIConfig.Configuration.SSM = &SSM{
			SsmUri:       "0.0.0.0:9000",
			AllowSsm:     false,
			TLS_Insecure: true,
			SsmSync: &SsmSync{
				Enable:           false,
				IntervalMinute:   0,
				MaxKeysCreate:    5,
				DeleteMissing:    false,
				MaxSyncKeys:      0,
				MaxSyncUsers:     0,
				MaxSyncRotations: 0,
			},
		}
	}
	if WebUIConfig.Configuration.SSM.SsmUri == "" {
		WebUIConfig.Configuration.SSM.SsmUri = "0.0.0.0:9000"
	}
	if WebUIConfig.Configuration.SSM.SsmSync == nil && WebUIConfig.Configuration.SSM.AllowSsm {
		logger.AppLog.Info("The ssm config is allow, but ssmsync is empty")
		WebUIConfig.Configuration.SSM.SsmSync = &SsmSync{
			Enable:           true,
			IntervalMinute:   60,
			MaxKeysCreate:    5,
			DeleteMissing:    true,
			MaxSyncKeys:      5,
			MaxSyncUsers:     5,
			MaxSyncRotations: 5,
		}
	}

	// Set defaults for Vault paths if missing
	if WebUIConfig.Configuration.Vault != nil {
		logger.AppLog.Info("The vault config is empty")
		v := WebUIConfig.Configuration.Vault
		if v.KeyKVPath == "" {
			v.KeyKVPath = "secret/data/k4keys"
		}
		if v.KeyKVMetadataPath == "" {
			v.KeyKVMetadataPath = "secret/metadata/k4keys"
		}
		if v.TransitKeysListPath == "" {
			v.TransitKeysListPath = "transit/keys"
		}
		if v.TransitKeyCreateFmt == "" {
			v.TransitKeyCreateFmt = "transit/keys/%s"
		}
		if v.TransitKeyRotateFmt == "" {
			v.TransitKeyRotateFmt = "transit/keys/%s/rotate"
		}
		if v.ConcurrencyOps == 0 {
			v.ConcurrencyOps = 10
		}
	}

	if WebUIConfig.Configuration.EnableAuthentication {
		if WebUIConfig.Configuration.Mongodb.WebuiDBName == "" ||
			WebUIConfig.Configuration.Mongodb.WebuiDBUrl == "" {
			return fmt.Errorf("[Configuration] if EnableAuthentication is set, WebuiDB must be set")
		}
	}

	if WebUIConfig.Configuration.Vault.AllowVault && WebUIConfig.Configuration.SSM.AllowSsm {
		return fmt.Errorf("[Configuration] SSM and Vault cannot be both enabled")
	}

	mongoConfig := WebUIConfig.Configuration.Mongodb
	if mongoConfig.DefaultConns == 0 {
		mongoConfig.DefaultConns = 500
	}
	if mongoConfig.AuthConns == 0 {
		mongoConfig.AuthConns = 100
	}
	if mongoConfig.WebuiDbConns == 0 {
		mongoConfig.WebuiDbConns = 100
	}
	if mongoConfig.ConcurrencyOps == 0 {
		mongoConfig.ConcurrencyOps = 30
	}

	return nil
}

func SetLogLevelsFromConfig(cfg *Config) {
	if cfg.Logger == nil {
		logger.InitLog.Warnln("webconsole config without log level setting")
		return
	}
	if cfg.Logger.WEBUI != nil {
		if cfg.Logger.WEBUI.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(cfg.Logger.WEBUI.DebugLevel); err != nil {
				logger.InitLog.Warnf("WebUI Log level [%s] is invalid, set to [info] level", cfg.Logger.WEBUI.DebugLevel)
				logger.SetLogLevel(zap.InfoLevel)
			} else {
				logger.InitLog.Infof("WebUI Log level is set to [%s] level", level)
				logger.SetLogLevel(level)
			}
		} else {
			logger.InitLog.Warnln("WebUI Log level not set. Default set to [info] level")
			logger.SetLogLevel(zap.InfoLevel)
		}
	}

	if cfg.Logger.MongoDBLibrary != nil {
		if cfg.Logger.MongoDBLibrary.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(cfg.Logger.MongoDBLibrary.DebugLevel); err != nil {
				utilLogger.AppLog.Warnf("MongoDBLibrary Log level [%s] is invalid, set to [info] level", cfg.Logger.MongoDBLibrary.DebugLevel)
				utilLogger.SetLogLevel(zap.InfoLevel)
			} else {
				utilLogger.SetLogLevel(level)
			}
		} else {
			utilLogger.AppLog.Warnln("MongoDBLibrary Log level not set. Default set to [info] level")
			utilLogger.SetLogLevel(zap.InfoLevel)
		}
	}
}

func GetCliFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "cfg",
			Usage: "Path to configuration file",
		},
	}
}
