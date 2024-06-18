// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !ui

package ui

import (
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

func AddUiService(engine *gin.Engine) *gin.RouterGroup {
	logger.WebUILog.Infoln("UI service will not be added")
	return nil
}
