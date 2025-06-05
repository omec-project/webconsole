// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

//go:build ui

package webui_service

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Webconsole API Documentation
// @version         1.0
// @contact.name    OMEC Project - Webconsole
// @contact.url     https://github.com/omec-project/webconsole
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html
// @host            localhost:5000
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Run the login endpoint to retrieve a token, then include its value in the format: `Bearer <token>`.
func AddSwaggerUiService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Adding Swagger UI service")
	endpoint := os.Getenv("WEBUI_ENDPOINT")
	if endpoint != "" {
		docs.SwaggerInfo.Host = endpoint
	}
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
