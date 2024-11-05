// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package webui_service

import (
	"bufio"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
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
	"github.com/omec-project/webconsole/dbadapter"
	gServ "github.com/omec-project/webconsole/proto/server"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type WEBUI struct{}

type (
	// Config information.
	Config struct {
		cfg string
	}
)

var config Config

var webuiCLi = []cli.Flag{
	cli.StringFlag{
		Name:     "cfg",
		Usage:    "webconsole config file",
		Required: true,
	},
}

func (*WEBUI) GetCliCmd() (flags []cli.Flag) {
	return webuiCLi
}

func (webui *WEBUI) Initialize(c *cli.Context) {
	config = Config{
		cfg: c.String("cfg"),
	}

	absPath, err := filepath.Abs(config.cfg)
	if err != nil {
		logger.ConfigLog.Errorln(err)
		return
	}

	if err := factory.InitConfigFactory(absPath); err != nil {
		logger.ConfigLog.Errorln(err)
		return
	}

	webui.setLogLevel()
}

func (webui *WEBUI) setLogLevel() {
	if factory.WebUIConfig.Logger == nil {
		logger.InitLog.Warnln("webconsole config without log level setting")
		return
	}

	if factory.WebUIConfig.Logger.WEBUI != nil {
		if factory.WebUIConfig.Logger.WEBUI.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(factory.WebUIConfig.Logger.WEBUI.DebugLevel); err != nil {
				logger.InitLog.Warnf("WebUI Log level [%s] is invalid, set to [info] level",
					factory.WebUIConfig.Logger.WEBUI.DebugLevel)
				logger.SetLogLevel(zap.InfoLevel)
			} else {
				logger.InitLog.Infof("WebUI Log level is set to [%s] level", level)
				logger.SetLogLevel(level)
			}
		} else {
			logger.InitLog.Warnln("WebUI Log level not set. Default set to [info] level")
			logger.SetLogLevel(zap.InfoLevel)
		}
	}

	if factory.WebUIConfig.Logger.MongoDBLibrary != nil {
		if factory.WebUIConfig.Logger.MongoDBLibrary.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(factory.WebUIConfig.Logger.MongoDBLibrary.DebugLevel); err != nil {
				utilLogger.AppLog.Warnf("MongoDBLibrary Log level [%s] is invalid, set to [info] level",
					factory.WebUIConfig.Logger.MongoDBLibrary.DebugLevel)
				utilLogger.SetLogLevel(zap.InfoLevel)
			} else {
				utilLogger.SetLogLevel(level)
			}
		} else {
			utilLogger.AppLog.Warnln("MongoDBLibrary Log level not set. Default set to [info] level")
			utilLogger.SetLogLevel(zap.InfoLevel)
		}
	}
}

func (webui *WEBUI) FilterCli(c *cli.Context) (args []string) {
	for _, flag := range webui.GetCliCmd() {
		name := flag.GetName()
		value := fmt.Sprint(c.Generic(name))
		if value == "" {
			continue
		}

		args = append(args, "--"+name, value)
	}
	return args
}

func setupAuthenticationFeature(subconfig_router *gin.Engine) {
	mongodb := factory.WebUIConfig.Configuration.Mongodb
	jwtSecret, err := auth.GenerateJWTSecret()
	if err != nil {
		logger.InitLog.Error(err)
	} else {
		dbadapter.ConnectMongo(mongodb.WebuiDBUrl, mongodb.WebuiDBName, &dbadapter.WebuiDBClient)
		resp, err := dbadapter.WebuiDBClient.CreateIndex(configmodels.UserAccountDataColl, "username")
		if !resp || err != nil {
			logger.InitLog.Errorf("error initializing webuiDB %v", err)
		}
		configapi.AddUserAccountService(subconfig_router, jwtSecret)
		auth.AddAuthenticationService(subconfig_router, jwtSecret)
		configapi.AddApiServiceWithAuthorization(subconfig_router, jwtSecret)
		configapi.AddConfigV1ServiceWithAuthorization(subconfig_router, jwtSecret)
	}
}

func (webui *WEBUI) Start() {
	// get config file info from WebUIConfig
	mongodb := factory.WebUIConfig.Configuration.Mongodb
	if factory.WebUIConfig.Configuration.Mode5G {
		// Connect to MongoDB
		dbadapter.ConnectMongo(mongodb.Url, mongodb.Name, &dbadapter.CommonDBClient)
		dbadapter.ConnectMongo(mongodb.AuthUrl, mongodb.AuthKeysDbName, &dbadapter.AuthDBClient)
	}

	logger.InitLog.Infoln("WebUI server started")

	/* First HTTP Server running at port to receive Config from ROC */
	subconfig_router := utilLogger.NewGinWithZap(logger.GinLog)
	if factory.WebUIConfig.Configuration.EnableAuthentication {
		setupAuthenticationFeature(subconfig_router)
	} else {
		configapi.AddApiService(subconfig_router)
		configapi.AddConfigV1Service(subconfig_router)
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
		tlsConfig := factory.WebUIConfig.Configuration.TLS
		if factory.WebUIConfig.Info.HttpVersion == 2 || tlsConfig != nil {
			server, err := http2_util.NewServer(httpAddr, "", subconfig_router)
			if server == nil {
				logger.InitLog.Errorln("initialize HTTP-2 server failed:", err)
				return
			}
			if err != nil {
				logger.InitLog.Warnln("initialize HTTP-2 server:", err)
				return
			}
			if tlsConfig != nil {
				err = server.ListenAndServeTLS(tlsConfig.PEM, tlsConfig.Key)
			} else {
				err = server.ListenAndServe()
			}
			if err != nil {
				logger.InitLog.Fatalln("HTTP server setup failed:", err)
				return
			}
		} else {
			logger.InitLog.Infoln(subconfig_router.Run(httpAddr))
			logger.InitLog.Infoln("Webserver stopped/terminated/not-started")
		}
	}()
	/* First HTTP server end */

	if factory.WebUIConfig.Configuration.Mode5G {
		self := webui_context.WEBUI_Self()
		self.UpdateNfProfiles()
	}

	// Start grpc Server. This has embedded functionality of sending
	// 4G config over REST Api as well.
	var host string = "0.0.0.0:9876"
	confServ := &gServ.ConfigServer{}
	go gServ.StartServer(host, confServ, configMsgChan)

	// fetch one time configuration from the simapp/roc on startup
	// this is to fetch existing config
	go fetchConfigAdapater()

	// http.ListenAndServe("0.0.0.0:5001", nil)

	select {}
}

func (webui *WEBUI) Exec(c *cli.Context) error {
	logger.InitLog.Debugln("args:", c.String("cfg"))
	args := webui.FilterCli(c)
	logger.InitLog.Debugln("filter:", args)
	command := exec.Command("webui", args...)

	webui.Initialize(c)

	stdout, err := command.StdoutPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			logger.InitLog.Infoln(in.Text())
		}
		wg.Done()
	}()

	stderr, err := command.StderrPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	go func() {
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			logger.InitLog.Infoln(in.Text())
		}
		wg.Done()
	}()

	go func() {
		if errCmd := command.Start(); errCmd != nil {
			logger.InitLog.Errorln("command.Start Failed")
		}
		wg.Done()
	}()

	wg.Wait()

	return err
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
