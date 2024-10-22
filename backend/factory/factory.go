// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
// Copyright 2024 Canonical Ltd
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

	"github.com/omec-project/webconsole/backend/logger"
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
