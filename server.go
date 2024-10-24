// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"os"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli"
)

var WEBUI = &webui_service.WEBUI{}

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	logger.AppLog.Infoln(app.Name)
	app.Usage = "-free5gccfg common configuration file -webuicfg webui configuration file"
	app.Action = action
	app.Flags = WEBUI.GetCliCmd()
	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Warnf("error args: %v", err)
	}
}

func action(c *cli.Context) {
	WEBUI.Initialize(c)
	WEBUI.Start()
}
