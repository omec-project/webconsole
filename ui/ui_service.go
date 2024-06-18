// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

// +build ui

 package ui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

//go:embed all:frontend_files
var FrontendFS embed.FS

func AddUiService(engine *gin.Engine) *gin.RouterGroup {
	group := engine.Group("/ui")
	logger.WebUILog.Infoln("Adding UI service")

	dist, err := fs.Sub(FrontendFS, "frontend_files")
	if err != nil {
		logger.WebUILog.Fatal(err)
		return nil
	}
	group.StaticFS("/", http.FS(dist))
	return group
}


