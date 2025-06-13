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
	gServ "github.com/omec-project/webconsole/configapi/server"
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
	"github.com/omec-project/webconsole/configmodels"
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

	configMsgChan := make(chan *configmodels.ConfigMessage, 10)
	configapi.SetChannel(configMsgChan)

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

	go func() {
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

		logger.InitLog.Infoln("Starting HTTP server on", httpAddr)
		if tlsConfig != nil {
			logger.InitLog.Infoln("Starting HTTPS server with TLS on", httpAddr)
			err = server.ListenAndServeTLS(tlsConfig.PEM, tlsConfig.Key)
		} else {
			logger.InitLog.Infoln("Starting HTTP server on", httpAddr)
			err = server.ListenAndServe()
		}
		if err != nil {
			logger.InitLog.Fatalln("HTTP server setup failed:", err)
		}
	}()

	if factory.WebUIConfig.Configuration.Mode5G {
		self := webui_context.WEBUI_Self()
		self.UpdateNfProfiles()
	}

	// Start grpc Server. This has embedded functionality of sending
	// 4G config over REST Api as well.
	host := "0.0.0.0:9876"
	confServ := &gServ.ConfigServer{}
	go gServ.StartServer(host, confServ, configMsgChan)

	// fetch one time configuration from the simapp/roc on startup
	// this is to fetch existing config
	go fetchConfigAdapater()

	// http.ListenAndServe("0.0.0.0:5001", nil)

	<-ctx.Done()
	logger.AppLog.Infoln("WebUI shutting down due to context cancel")
}

func fetchConfigAdapater() {
	for {
		if (factory.WebUIConfig.Configuration == nil) ||
			(factory.WebUIConfig.Configuration.RocEnd == nil) ||
			(!factory.WebUIConfig.Configuration.RocEnd.Enabled) ||
			(factory.WebUIConfig.Configuration.RocEnd.SyncUrl == "") {
			time.Sleep(1 * time.Second)
			continue
		}

		client := &http.Client{}
		httpend := factory.WebUIConfig.Configuration.RocEnd.SyncUrl
		req, err := http.NewRequest(http.MethodPost, httpend, nil)
		// Handle Error
		if err != nil {
			logger.InitLog.Errorf("an error occurred %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		// set the request header Content-Type for json
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(req)
		if err != nil {
			logger.InitLog.Errorf("an error occurred %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			logger.InitLog.Errorf("an error occurred %v", err)
		}
		logger.InitLog.Infof("fetching config from simapp/roc. Response code = %d", resp.StatusCode)
		break
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
