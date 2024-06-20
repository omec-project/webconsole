// SPDX-License-Identifier: Apache-2.0

// +build !ui

package webui_service

import (
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

func AddUiService(engine *gin.Engine) *gin.RouterGroup {
	logger.WebUILog.Infoln("UI service will not be added")
	return nil
}
