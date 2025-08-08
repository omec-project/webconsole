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
