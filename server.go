// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli"
	"os"
)

var (
	WEBUI = &webui_service.WEBUI{}
)

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "Web UI"
	app.UsageText = "webconsole -cfg <webui_config_file.conf>"
	app.Action = func(c *cli.Context) error {
		return action(c)
	}
	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Errorf("error args: %v", err)
	}
}

func action(c *cli.Context) error {
	config, err := WEBUI.Initialize(c)
	if err != nil {
		logger.AppLog.Errorf("Failed to initialize WEBUI: %v", err)
		return err
	}

	nf, err := nfconfig.NewNFConfigFunc(config)
	if err != nil {
		logger.AppLog.Errorf("Failed to create NFConfig: %v", err)
		return err
	}

	return runWebUIAndNFConfig(WEBUI, nf)
}

func runWebUIAndNFConfig(webui webui_service.WebUIInterface, nf nfconfig.NFConfigInterface) error {
	errChan := make(chan error, 1)
	logger.InitLog.Infoln("Starting NFConfig")
	go func() {
		if err := nf.Start(); err != nil {
			logger.InitLog.Errorf("NFConfig start failed: %v", err)
			errChan <- err
		}
		logger.InitLog.Infoln("Started NFConfig")
	}()

	go webui.Start()

	select {
	case err := <-errChan:
		logger.InitLog.Errorf("NFConfig server failed: %v", err)
		return err
	}
}
