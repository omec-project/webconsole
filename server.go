// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/omec-project/webconsole/dbadapter"
	"github.com/urfave/cli"
)

var (
	initMongoDB       = dbadapter.InitMongoDB
	newNFConfigServer = nfconfig.NewNFConfigServer
	runServer         = runWebUIAndNFConfig
)

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "Web UI"
	app.UsageText = "webconsole -cfg <webui_config_file.yaml>"
	app.Flags = factory.GetCliFlags()
	app.Action = func(c *cli.Context) error {
		cfgPath := c.String("cfg")
		if cfgPath == "" {
			return fmt.Errorf("required flag cfg not set")
		}

		absPath, err := filepath.Abs(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}

		if err := factory.InitConfigFactory(absPath); err != nil {
			return fmt.Errorf("failed to init config: %w", err)
		}

		config := factory.WebUIConfig
		if config == nil {
			return fmt.Errorf("configuration not properly initialized")
		}
		factory.SetLogLevelsFromConfig(config)

		return startApplication(config)
	}

	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Fatalf("error args: %v", err)
	}
}

func startApplication(config *factory.Config) error {
	if config == nil || config.Configuration == nil {
		return fmt.Errorf("configuration section is nil")
	}
	if err := initMongoDB(); err != nil {
		logger.InitLog.Errorf("failed to initialize MongoDB: %v", err)
		return err
	}

	nfConfigServer, err := newNFConfigServer(config)
	if err != nil {
		return fmt.Errorf("failed to initialize NFConfig: %w", err)
	}
	webui := &webui_service.WEBUI{
		TriggerSyncNFConfigFunc: nfConfigServer.TriggerSync,
	}
	return runServer(webui, nfConfigServer)
}

func runWebUIAndNFConfig(webui webui_service.WebUIInterface, nfConf nfconfig.NFConfigInterface) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go webui.Start(ctx)
	logger.InitLog.Infoln("WebUI started")

	err := nfConf.Start(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("NFConfig failed: %w", err)
	}

	return nil
}
