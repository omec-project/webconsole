// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/free5gc/version"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/webui_service"
)

var WEBUI = &webui_service.WEBUI{}

var appLog *logrus.Entry

func init() {
	appLog = logger.AppLog
}

func main() {
	app := cli.NewApp()
	app.Name = "webui"
	appLog.Infoln(app.Name)
	appLog.Infoln("webconsole version: ", version.GetVersion())
	app.Usage = "-free5gccfg common configuration file -webuicfg webui configuration file"
	app.Action = action
	app.Flags = WEBUI.GetCliCmd()
	if err := app.Run(os.Args); err != nil {
		logger.AppLog.Warnf("Error args: %v", err)
	}
}

func action(c *cli.Context) {
	WEBUI.Initialize(c)
	WEBUI.Start()
}
