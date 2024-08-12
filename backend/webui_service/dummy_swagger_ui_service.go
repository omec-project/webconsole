// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd

//go:build !ui

package webui_service

import (
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

func AddSwaggerUiService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Swagger UI service will not be added")
}
