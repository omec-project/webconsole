// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

// +build ui

package dynamic_parameter_service

import (
	"fmt"
    "io/ioutil"
    "errors"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/omec-project/webconsole/backend/logger"
)

type route struct {
    Pattern string
	EnvVariable string
}

func AddDynamicParameterService(engine *gin.Engine) {
	logger.WebUILog.Infoln("Adding dynamic parameters service")
    group := engine.Group("/config/parameter")
    for _, parameter := range dynamicParameters {
        serveDynamicParameter(group, parameter.Pattern, parameter.EnvVariable);
	}
}

func serveDynamicParameter(group *gin.RouterGroup, pattern string, envVariable string){
    fileContent, err := readFileFromEnvVariable(envVariable)
	if err != nil {
        logger.WebUILog.Warningf("/config/parameter%s route will not be served", pattern)
		return
	}
	group.GET(pattern, func(c *gin.Context) {
        c.String(200, "%s", fileContent)
    })
}

func readFileFromEnvVariable(envVariable string) ([]byte, error) {
	filePath := os.Getenv(envVariable)
    if filePath == "" {
		err := errors.New(fmt.Sprintf("Environment variable %s is not set", envVariable))
        logger.WebUILog.Warningln(err)
		return nil, err
    }

    fileContent, fileContentErr := ioutil.ReadFile(filePath)
    if fileContentErr != nil {
        logger.WebUILog.Warningf("Failed to read the file: %v", fileContentErr)
		return nil, fileContentErr
    }
	return fileContent, nil;
}

var dynamicParameters = []route{
	{
        "/gnb",
		"GNB_CONFIG_PATH",
	},
	{
        "/upf",
		"UPF_CONFIG_PATH",
	},
}
