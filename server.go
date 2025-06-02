// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"github.com/omec-project/webconsole/backend/factory"
	"os"
	"path/filepath"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli"
)

var WEBUI = &webui_service.WEBUI{}

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "Web UI"
	app.UsageText = "webconsole -cfg <webui_config_file.yaml>"
	app.Flags = WEBUI.GetCliCmd()
	app.Action = func(c *cli.Context) error {
		cfgPath := c.String("cfg")
		if cfgPath == "" {
			return fmt.Errorf("required flag cfg not set")
		}

		absPath, err := filepath.Abs(cfgPath)
		if err != nil {
			logger.ConfigLog.Errorln(err)
			return err
		}

		if err := factory.InitConfigFactory(absPath); err != nil {
			logger.ConfigLog.Errorln(err)
			return err
		}
		config := factory.WebUIConfig
		factory.SetLogLevelsFromConfig(config)

		WEBUI := &webui_service.WEBUI{}
		nf, err := nfconfig.NewNFConfigFunc(config)
		if err != nil {
			logger.AppLog.Errorf("Failed to create NFConfig: %v", err)
			return err
		}
		return runWebUIAndNFConfig(WEBUI, nf)
	}

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("error args: %v", err)
	}

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("error args: %v", err)
	}
}

func runWebUIAndNFConfig(webui webui_service.WebUIInterface, nf nfconfig.NFConfigInterface) error {
	errChan := make(chan error, 2)
	go func() {
		err := nf.Start()
		errChan <- err
	}()

	go func() {
		webui.Start()
	}()

	if err := <-errChan; err != nil {
		logger.AppLog.Errorf("Service exited with error: %v", err)
		return err
	}
	return nil
}
