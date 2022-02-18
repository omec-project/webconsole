// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * WebUI Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

var WebUIConfig *Config

func init() {
	WebUIConfig = &Config{}
}

func GetConfig() *Config {
	return WebUIConfig
}

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	} else {
		if yamlErr := yaml.Unmarshal(content, WebUIConfig); yamlErr != nil {
			return fmt.Errorf("[Configuration] %+v", yamlErr)
		}
		// we dont want Mode5G coming from the helm chart, since
		// there is chance of misconfiguration
		if os.Getenv("CONFIGPOD_DEPLOYMENT") == "4G" {
			fmt.Println("ConfigPod running in 4G deployment")
			WebUIConfig.Configuration.Mode5G = false
		} else {
			//default mode
			fmt.Println("ConfigPod running in 5G deployment")
			WebUIConfig.Configuration.Mode5G = true
		}
	}

	return nil
}
