// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

// +build ui

package webui_service

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title		Webconsole API Documentation
//	@version	1.0

//	@contact.name	OMEC Project - Webconsole
//	@contact.url	https://github.com/omec-project/webconsole

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host		localhost:5000
// @BasePath	/
func AddSwaggerUiService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Adding Swagger UI service")
	host := os.Getenv("SWAGGER_HOST")
    if host != "" {
		docs.SwaggerInfo.Host = host + ":5000"
		logger.WebUILog.Infoln(docs.SwaggerInfo.Host)
    }
    engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
