// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

//go:build !ui

package dynamic_parameter_service

import (
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

func AddDynamicParameterService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Dynamic parameters service will not be added")
}