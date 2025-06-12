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

type NFConfigServer struct {
	Config *factory.Configuration
	Router *gin.Engine
}

type Route struct {
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type NFConfigInterface interface {
	Start(ctx context.Context) error
}

func (n *NFConfigServer) router() *gin.Engine {
	return n.Router
}

func enforceAcceptJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader != "application/json" {
			logger.ConfigLog.Infoln("Invalid Accept header value: '%s'. Expected 'application/json'", acceptHeader)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Accept header must be 'application/json'",
			})
			return
		}
		c.Next()
	}
}

func NewNFConfigServer(config *factory.Config) (NFConfigInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(enforceAcceptJSON())

	nfconfigServer := &NFConfigServer{
		Config: config.Configuration,
		Router: router,
	}
	logger.InitLog.Infoln("Setting up NFConfig routes")
	nfconfigServer.setupRoutes()
	return nfconfigServer, nil
}

func (n *NFConfigServer) Start(ctx context.Context) error {
	addr := ":5001"
	srv := &http.Server{
		Addr:    addr,
		Handler: n.Router,
	}
	serverErrChan := make(chan error, 1)
	go func() {
		if n.Config.NfConfigTLS != nil && n.Config.NfConfigTLS.Key != "" && n.Config.NfConfigTLS.PEM != "" {
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

func (n *NFConfigServer) setupRoutes() {
	api := n.Router.Group("/nfconfig")
	for _, route := range n.getRoutes() {
		api.GET(route.Pattern, route.HandlerFunc)
	}
}

func (n *NFConfigServer) getRoutes() []Route {
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
