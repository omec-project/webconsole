// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"os"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli"
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
	app.Action = action
	app.Flags = WEBUI.GetCliCmd()
	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("error args: %v", err)
	}
}

func action(c *cli.Context) {
	config, err := WEBUI.Initialize(c)
	if err != nil {
		logger.InitLog.Errorf("Failed to initialize WEBUI: %v", err)
		return
	}

	nfConfig, err := nfconfig.NewNFConfigFunc(config)
	if err != nil {
		logger.InitLog.Errorf("Failed to create NFConfig: %v", err)
		return
	}
	go WEBUI.Start()
	errChan := make(chan error, 1)
	go func() {
		errChan <- nfConfig.Start()
	}()

	if err := <-errChan; err != nil {
		logger.InitLog.Errorf("NFConfig Service error: %v", err)
		return
	}
}
