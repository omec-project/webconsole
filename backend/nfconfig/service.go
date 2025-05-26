// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"net/http"
	"os"
)

type NFConfig struct {
	router *gin.Engine
	cfg    ServiceConfiguration
}

func NewNFConfig() *NFConfig {
	return &NFConfig{
		router: gin.Default(),
	}
}

func (n *NFConfig) loadConfig(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &n.cfg); err != nil {
		return err
	}

	_, keyErr := os.Stat(n.cfg.TLS.Key)
	_, pemErr := os.Stat(n.cfg.TLS.Pem)
	if keyErr != nil || pemErr != nil {
		// One or two files don't exist, disable TLS
		n.cfg.TLS.enabled = false
		return nil
	}

	// Both files exist, enable TLS
	n.cfg.TLS.enabled = true
	return nil
}

func (n *NFConfig) GetCliCmd() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "nfconfig-cfg",
			Usage: "Path to NFConfig configuration file",
		},
	}
}

func (n *NFConfig) Initialize(c *cli.Context) error {
	configPath := c.String("nfconfig-cfg")
	if configPath == "" {
		return fmt.Errorf("configuration file not specified")
	}

	if err := n.loadConfig(configPath); err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	n.setupRoutes()
	return nil
}

func (n *NFConfig) Start() error {
	addr := fmt.Sprintf(":%d", 9090)
	srv := &http.Server{
		Addr:    addr,
		Handler: n.router,
	}

	if n.cfg.TLS.enabled {
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
