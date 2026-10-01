// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package webui_service

import (
	"context"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/http2_util"
	utilLogger "github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/auth"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/metrics"
	"github.com/omec-project/webconsole/backend/webui_context"
	"github.com/omec-project/webconsole/configapi"
)

type WEBUI struct{}

type WebUIInterface interface {
	Start(ctx context.Context, syncChan chan<- struct{})
}

func setupAuthenticationFeature(subconfig_router *gin.Engine, nfSyncMiddelware gin.HandlerFunc) {
	jwtSecret, err := auth.GenerateJWTSecret()
	if err != nil {
		logger.InitLog.Error(err)
		return
	}
	configapi.AddUserAccountService(subconfig_router, jwtSecret)
	auth.AddAuthenticationService(subconfig_router, jwtSecret)
	authMiddleware := auth.AdminOrUserAuthMiddleware(jwtSecret)
	configapi.AddApiService(subconfig_router, authMiddleware)
	configapi.AddConfigV1Service(subconfig_router, nfSyncMiddelware, authMiddleware)
}

func (webui *WEBUI) Start(ctx context.Context, syncChan chan<- struct{}) {
	subconfig_router := utilLogger.NewGinWithZap(logger.GinLog)
	nFConfigSyncMiddleware := triggerNFConfigSyncMiddleware(syncChan)
	if factory.WebUIConfig.Configuration.EnableAuthentication {
		setupAuthenticationFeature(subconfig_router, nFConfigSyncMiddleware)
	} else {
		configapi.AddApiService(subconfig_router)
		configapi.AddConfigV1Service(subconfig_router, nFConfigSyncMiddleware)
	}
	AddSwaggerUiService(subconfig_router)
	AddUiService(subconfig_router)

	go metrics.InitMetrics()

	subconfig_router.Use(cors.New(cors.Config{
		AllowMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin", "Content-Length", "Content-Type", "User-Agent",
			"Referrer", "Host", "Token", "X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  true,
		MaxAge:           86400,
	}))

	// The goroutine sends the configured server, or closes the channel without
	// sending when configuration fails, so the shutdown below knows which.
	serverCh := make(chan *http.Server, 1)
	go func() {
		defer close(serverCh)
		httpAddr := ":" + strconv.Itoa(factory.WebUIConfig.Configuration.CfgPort)
		logger.InitLog.Infoln("Webui HTTP addr", httpAddr)
		tlsConfig := factory.WebUIConfig.Configuration.WebuiTLS
		var server *http.Server
		var err error
		if factory.WebUIConfig.Info.HttpVersion == 2 {
			logger.InitLog.Infoln("Configuring HTTP/2 server...")
			server, err = http2_util.NewServer(httpAddr, "", subconfig_router)
			if server == nil {
				logger.InitLog.Errorln("initialize HTTP-2 server failed:", err)
				return
			}
			if err != nil {
				logger.InitLog.Warnln("initialize HTTP-2 server:", err)
				return
			}
			logger.InitLog.Infoln("HTTP/2 server configured successfully")
		} else {
			logger.InitLog.Infoln("Configuring HTTP/1.1 server...")
			server = &http.Server{
				Addr:    httpAddr,
				Handler: subconfig_router,
			}
		}

		serverCh <- server
		if tlsConfig != nil {
			logger.InitLog.Infoln("Starting HTTPS server with TLS on", httpAddr)
			err = server.ListenAndServeTLS(tlsConfig.PEM, tlsConfig.Key)
		} else {
			logger.InitLog.Infoln("Starting HTTP server on", httpAddr)
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.InitLog.Fatalln("HTTP server setup failed:", err)
		}
	}()

	self := webui_context.WEBUI_Self()
	self.UpdateNfProfiles()

	<-ctx.Done()
	logger.AppLog.Infoln("WebUI shutting down due to context cancel")
	// Wait for configuration to end either way: a server that is still being
	// configured would otherwise start listening after Start has returned.
	// Shutdown also works before ListenAndServe, which then returns at once.
	server, ok := <-serverCh
	if !ok {
		// Configuration failed; there is no server to shut down.
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.AppLog.Warnln("WebUI server shutdown:", err)
	}
}

func triggerNFConfigSyncMiddleware(syncChan chan<- struct{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if isWritingMethod(c.Request.Method) && isStatusSuccess(c.Writer.Status()) {
			syncChan <- struct{}{}
			logger.WebUILog.Infoln("NF config sync triggered via middleware")
		} else {
			logger.WebUILog.Debugln("WebUI operation does not require NF configuration synchronization")
		}
	}
}

func isWritingMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodDelete || method == http.MethodPatch
}

func isStatusSuccess(status int) bool {
	return status/100 == 2
}
