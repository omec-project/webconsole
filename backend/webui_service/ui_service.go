// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

//go:build ui

package webui_service

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/ui"
)

func AddUiService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Adding UI service")
	staticFilesSystem, err := fs.Sub(ui.FrontendFS, "frontend_files")
	if err != nil {
		logger.WebUILog.Fatal(err)
	}

	engine.Use(func(c *gin.Context) {
		if !isApiUrlPath(c.Request.URL.Path, c.Request.Method) {
			htmlPath := strings.TrimPrefix(c.Request.URL.Path, "/") + ".html"
			if _, err := staticFilesSystem.Open(htmlPath); err == nil {
				c.Request.URL.Path = htmlPath
			}
			fileServer := http.FileServer(http.FS(staticFilesSystem))
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		}
	})
}

func isApiUrlPath(path string, method string) bool {
	if path == "/login" && method == http.MethodPost {
		return true
	}
	if path == "/status" && method == http.MethodGet {
		return true
	}
	return strings.HasPrefix(path, "/config/v1/") || strings.HasPrefix(path, "/api/")
}
