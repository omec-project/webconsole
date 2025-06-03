// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "Web UI"
	app.UsageText = "webconsole -cfg <webui_config_file.yaml>"
	tempWEBUI := &webui_service.WEBUI{}
	app.Flags = tempWEBUI.GetCliCmd()
	app.Action = func(c *cli.Context) error {
		cfgPath := c.String("cfg")
		if cfgPath == "" {
			return fmt.Errorf("required flag cfg not set")
		}

		absPath, pathErr := filepath.Abs(cfgPath)
		if pathErr != nil {
			logger.ConfigLog.Errorln(pathErr)
			return pathErr
		}

		if err := factory.InitConfigFactory(absPath); err != nil {
			logger.ConfigLog.Errorln(err)
			return err
		}
		config := factory.WebUIConfig
		if config == nil {
			return fmt.Errorf("configuration not properly initialized")
		}
		factory.SetLogLevelsFromConfig(config)

		webui := &webui_service.WEBUI{}
		nfConf, err := nfconfig.NewNFConfigFunc(config)
		if err != nil {
			logger.AppLog.Errorf("Failed to create NFConfig: %v", err)
			return err
		}
		return runWebUIAndNFConfig(webui, nfConf)
	}

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("error args: %v", err)
	}
}

func runWebUIAndNFConfig(webui webui_service.WebUIInterface, nfConf nfconfig.NFConfigInterface) error {
	go func() {
		webui.Start()
	}()

	if err := nfConf.Start(); err != nil {
		logger.AppLog.Errorf("Service exited with error: %v", err)
		return err
	}
	return nil
}
