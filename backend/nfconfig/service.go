// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type NFConfigServer struct {
	config         *factory.Configuration
	Router         *gin.Engine
	inMemoryConfig inMemoryConfig
	syncCancelFunc context.CancelFunc
	syncMutex      sync.Mutex
}

type Route struct {
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type NFConfigInterface interface {
	Start(ctx context.Context, syncChan <-chan struct{}) error
}

func (n *NFConfigServer) router() *gin.Engine {
	return n.Router
}

func NewNFConfigServer(config *factory.Config) (NFConfigInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(enforceAcceptJSON())

	nfconfigServer := &NFConfigServer{
		config: config.Configuration,
		Router: router,
	}

	if err := nfconfigServer.syncInMemoryConfig(); err != nil {
		return nil, fmt.Errorf("failed to sync NF configuration data: %w", err)
	}

	logger.InitLog.Infoln("Setting up NFConfig routes")
	nfconfigServer.setupRoutes()
	return nfconfigServer, nil
}

func (n *NFConfigServer) Start(ctx context.Context, syncChan <-chan struct{}) error {
	n.startSyncWorker(ctx, syncChan)
	addr := ":5001"
	srv := &http.Server{
		Addr:    addr,
		Handler: n.Router,
	}
	serverErrChan := make(chan error, 1)
	go func() {
		if n.config.NfConfigTLS != nil && n.config.NfConfigTLS.Key != "" && n.config.NfConfigTLS.PEM != "" {
			logger.NfConfigLog.Infoln("Starting HTTPS server on", addr)
			serverErrChan <- srv.ListenAndServeTLS(n.config.NfConfigTLS.PEM, n.config.NfConfigTLS.Key)
		} else {
			logger.NfConfigLog.Infoln("Starting HTTP server on", addr)
			serverErrChan <- srv.ListenAndServe()
		}
	}()
	select {
	case <-ctx.Done():
		logger.NfConfigLog.Infoln("NFConfig context cancelled, shutting down server.")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)

	case err := <-serverErrChan:
		return err
	}
}

func (n *NFConfigServer) startSyncWorker(ctx context.Context, syncChan <-chan struct{}) {
	go func() {
		var currentCancel context.CancelFunc

		for {
			select {
			case <-ctx.Done():
				if currentCancel != nil {
					currentCancel()
				}
				return

			case <-syncChan:
				// Cancel current sync if running
				if currentCancel != nil {
					logger.NfConfigLog.Infoln("Cancelling ongoing sync due to new trigger")
					currentCancel()
				}

				var syncCtx context.Context
				syncCtx, currentCancel = context.WithCancel(context.Background())
				go n.syncWithRetry(syncCtx)
			}
		}
	}()
}

func (n *NFConfigServer) syncWithRetry(ctx context.Context) {
	n.syncMutex.Lock()
	defer n.syncMutex.Unlock()
	logger.NfConfigLog.Debugln("Starting in-memory NF configuration synchronization with new context")

	for {
		select {
		case <-ctx.Done():
			logger.NfConfigLog.Infoln("No-op. Sync in-memory configuration was cancelled")
			return
		default:
			err := syncInMemoryConfigFunc(n)
			if err == nil {
				return
			}
			logger.NfConfigLog.Warnf("Sync in-memory configuration failed, retrying: %v", err)
			time.Sleep(3 * time.Second)
		}
	}
}

var syncInMemoryConfigFunc = func(n *NFConfigServer) error {
	return n.syncInMemoryConfig()
}

func (n *NFConfigServer) syncInMemoryConfig() error {
	sliceDataColl := "webconsoleData.snapshots.sliceData"
	rawSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, bson.M{})
	if err != nil {
		return err
	}

	slices := []configmodels.Slice{}
	for _, rawSlice := range rawSlices {
		var s configmodels.Slice
		if err := json.Unmarshal(configmodels.MapToByte(rawSlice), &s); err != nil {
			logger.NfConfigLog.Warnf("Failed to unmarshal slice: %v. Network slice `%s` will be ignored", err, s.SliceName)
			continue
		}
		slices = append(slices, s)
	}
	logger.NfConfigLog.Debugf("Retrieved %d network slices", len(slices))
	n.inMemoryConfig.syncPlmn(slices)
	n.inMemoryConfig.syncPlmnSnssai(slices)
	n.inMemoryConfig.syncAccessAndMobility()
	n.inMemoryConfig.syncSessionManagement()
	n.inMemoryConfig.syncPolicyControl()
	logger.NfConfigLog.Infoln("Updated NF in-memory configuration")
	return nil
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

func enforceAcceptJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader != "application/json" {
			logger.NfConfigLog.Warnf("Invalid Accept header value: '%s'. Expected 'application/json'", acceptHeader)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Accept header must be 'application/json'",
			})
			return
		}
		c.Next()
	}
}
