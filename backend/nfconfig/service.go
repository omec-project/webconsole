// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
)

type NFConfig struct {
	Config *factory.Configuration
	router *gin.Engine
}

type Route struct {
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type NFConfigInterface interface {
	Start(ctx context.Context) error
}

func (n *NFConfig) Router() *gin.Engine {
	return n.router
}

var NewNFConfigFunc = NewNFConfig

func NewNFConfig(config *factory.Config) (NFConfigInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	gin.SetMode(gin.ReleaseMode)
	nfconfig := &NFConfig{
		Config: config.Configuration,
		router: gin.Default(),
	}
	logger.InitLog.Infoln("Setting up NFConfig routes")
	nfconfig.setupRoutes()
	return nfconfig, nil
}

func (n *NFConfig) Start(ctx context.Context) error {
	addr := ":9090"
	srv := &http.Server{
		Addr:    addr,
		Handler: n.router,
	}
	serverErrChan := make(chan error, 1)
	go func() {
		if n.Config.NfConfigTLS.Key != "" && n.Config.NfConfigTLS.PEM != "" {
			logger.ConfigLog.Infoln("Starting HTTPS server on", addr)
			serverErrChan <- srv.ListenAndServeTLS(n.Config.NfConfigTLS.PEM, n.Config.NfConfigTLS.Key)
		} else {
			logger.ConfigLog.Infoln("Starting HTTP server on", addr)
			serverErrChan <- srv.ListenAndServe()
		}
	}()
	select {
	case <-ctx.Done():
		logger.ConfigLog.Infoln("NFConfig context cancelled, shutting down server.")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)

	case err := <-serverErrChan:
		return err
	}
}

func (n *NFConfig) setupRoutes() {
	api := n.router.Group("/nfconfig")
	for _, route := range n.getRoutes() {
		api.GET(route.Pattern, route.HandlerFunc)
	}
}

func (n *NFConfig) getRoutes() []Route {
	return []Route{
		{
			Pattern:     "/access-mobility",
			HandlerFunc: n.GetAccessMobilityConfig,
		},
		{
			Pattern:     "/plmn",
			HandlerFunc: n.GetPlmnConfig,
		},
		{
			Pattern:     "/plmn-snssai",
			HandlerFunc: n.GetPlmnSnssaiConfig,
		},
		{
			Pattern:     "/policy-control",
			HandlerFunc: n.GetPolicyControlConfig,
		},
		{
			Pattern:     "/session-management",
			HandlerFunc: n.GetSessionManagementConfig,
		},
	}
}
