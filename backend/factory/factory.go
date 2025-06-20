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
	if content, err := os.ReadFile(f); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	} else {
		if yamlErr := yaml.Unmarshal(content, WebUIConfig); yamlErr != nil {
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

		if WebUIConfig.Configuration.EnableAuthentication {
			if WebUIConfig.Configuration.Mongodb.WebuiDBName == "" ||
				WebUIConfig.Configuration.Mongodb.WebuiDBUrl == "" {
				return fmt.Errorf("[Configuration] if EnableAuthentication is set, WebuiDB must be set")
			}
		}
		// we dont want Mode5G coming from the helm chart, since
		// there is chance of misconfiguration
		if os.Getenv("CONFIGPOD_DEPLOYMENT") == "4G" {
			logger.ConfigLog.Infoln("configPod running in 4G deployment")
			WebUIConfig.Configuration.Mode5G = false
		} else {
			// default mode
			logger.ConfigLog.Infoln("configPod running in 5G deployment")
			WebUIConfig.Configuration.Mode5G = true
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
