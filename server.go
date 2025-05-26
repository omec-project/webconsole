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
	"sync"
)

var (
	WEBUI    = &webui_service.WEBUI{}
	NFConfig = nfconfig.NewNFConfig()
)

func main() {
	app := cli.NewApp()
	app.Name = "webconsole"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "Web Console and NF Configuration Service"
	app.UsageText = "webconsole -cfg <webui_config_file.conf> -nfconfig-cfg <nfconfig_config_file.conf>"
	app.Action = action

	app.Flags = append(WEBUI.GetCliCmd(), NFConfig.GetCliCmd()...)

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("Error running application: %v", err)
	}
}

func action(c *cli.Context) error {
	logger.AppLog.Infoln("Initializing services...")

	if err := WEBUI.Initialize(c); err != nil {
		logger.AppLog.Fatalf("Failed to initialize WEBUI: %v", err)
	}

	if err := NFConfig.Initialize(c); err != nil {
		logger.AppLog.Fatalf("Failed to initialize NFConfig: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Start WEBUI service
	go func() {
		defer wg.Done()
		logger.AppLog.Infoln("Starting WEBUI service...")
		WEBUI.Start()
	}()

	// Start NFConfig service
	go func() {
		defer wg.Done()
		logger.AppLog.Infoln("Starting NFConfig service...")
		err := NFConfig.Start()
		if err != nil {
			logger.AppLog.Errorf("NFConfig service error: %v", err)
		}
	}()

	wg.Wait()
	logger.AppLog.Infoln("All services stopped")
	return nil
}
