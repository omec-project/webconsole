// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
)

type NFConfig struct {
	router *gin.Engine
	cfg    ServiceConfiguration
}

type NFConfigFactory struct {
	configPath string
	cfg        ServiceConfiguration
}

func NewNFConfigFactory(configPath string) *NFConfigFactory {
	return &NFConfigFactory{
		configPath: configPath,
	}
}

func (f *NFConfigFactory) Create() (*NFConfig, error) {
	if f.configPath == "" {
		return nil, fmt.Errorf("configuration file path not specified")
	}

	data, err := os.ReadFile(f.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &f.cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	if f.cfg.TLS.Enabled {
		logger.ConfigLog.Infof("Checking TLS files - Key: %s, Pem: %s", f.cfg.TLS.Key, f.cfg.TLS.Pem)
		_, keyErr := os.Stat(f.cfg.TLS.Key)
		_, pemErr := os.Stat(f.cfg.TLS.Pem)
		if keyErr != nil || pemErr != nil {
			logger.ConfigLog.Errorf("TLS file check failed - KeyErr: %v, PemErr: %v", keyErr, pemErr)
			f.cfg.TLS.Enabled = false
		} else {
			logger.ConfigLog.Info("TLS files found successfully")
		}
	}

	nfConfig := &NFConfig{
		router: gin.Default(),
		cfg:    f.cfg,
	}

	nfConfig.setupRoutes()
	return nfConfig, nil
}

func (n *NFConfig) Start() error {
	addr := fmt.Sprintf(":%d", 9090)
	srv := &http.Server{
		Addr:    addr,
		Handler: n.router,
	}

	if n.cfg.TLS.Enabled {
		logger.ConfigLog.Infoln("Starting HTTPS server on", addr)
		return srv.ListenAndServeTLS(n.cfg.TLS.Pem, n.cfg.TLS.Key)
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
