// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
)

type NFConfig struct {
	Config *factory.Configuration
	router *gin.Engine
}

type NFConfigInterface interface {
	Start() error
}

var NewNFConfigFunc = NewNFConfig

func NewNFConfig(config *factory.Config) (NFConfigInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if config.Logger.WEBUI != nil {
		if config.Logger.WEBUI.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(config.Logger.WEBUI.DebugLevel); err != nil {
				logger.InitLog.Warnf("NFConfig Log level [%s] is invalid, set to [info] level",
					config.Logger.WEBUI.DebugLevel)
				logger.SetLogLevel(zap.InfoLevel)
			} else {
				logger.InitLog.Infof("NFConfig Log level is set to [%s] level", level)
				logger.SetLogLevel(level)
			}
		} else {
			logger.InitLog.Warnln("NFConfig Log level not set. Default set to [info] level")
			logger.SetLogLevel(zap.InfoLevel)
		}
	}

	nf := &NFConfig{
		Config: config.Configuration,
		router: gin.Default(),
	}
	nf.setupRoutes()
	err := nf.Start()
	if err != nil {
		logger.InitLog.Errorf("NFConfig Start failed: %v", err)
		return nil, err
	}

	return nf, nil
}

func (n *NFConfig) Start() error {
	addr := fmt.Sprintf(":%d", 9090)
	srv := &http.Server{
		Addr:    addr,
		Handler: n.router,
	}

	if n.Config.ConfigTLS.Key != "" && n.Config.ConfigTLS.PEM != "" {
		logger.ConfigLog.Infoln("Starting HTTPS server on", addr)
		return srv.ListenAndServeTLS(n.Config.ConfigTLS.PEM, n.Config.ConfigTLS.Key)
	}

	logger.ConfigLog.Infoln("Starting HTTP server on", addr)
	return srv.ListenAndServe()
}

func (n *NFConfig) setupRoutes() {
	api := n.router.Group("/nfconfig")
	{
		api.GET("/access-mobility", n.GetAccessMobilityConfig)
		api.GET("/plmn", n.GetPlmnConfig)
		api.GET("/plmn-snssai", n.GetPlmnSnssaiConfig)
		api.GET("/policy-control", n.GetPolicyControlConfig)
		api.GET("/session-management", n.GetSessionManagementConfig)
	}
}
