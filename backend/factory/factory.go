// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/*
 * WebUI Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var WebUIConfig Config

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Configuration] %+v", err)
	} else {
		WebUIConfig = Config{}

		if yamlErr := yaml.Unmarshal(content, &WebUIConfig); yamlErr != nil {
			return fmt.Errorf("[Configuration] %+v", yamlErr)
		}
	}

	return nil
}
