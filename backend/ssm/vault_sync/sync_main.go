// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package vaultsync

import (
	"errors"
	"sync"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
)

var (
	// ErrorSyncChan channel for synchronization errors
	// ErrorSyncChan chan error = make(chan error, 10)

	// StopVaultSyncFunction flag to stop synchronization
	StopVaultSyncFunction bool = false

	// healthMutex for thread-safe access to StopVaultSyncFunction
	healthMutex sync.Mutex
)

const (
	internalKeyLabel = "aes256-gcm"
)

// getTransitKeysListPath returns the transit keys list path from configuration
func getTransitKeysListPath() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if path := factory.WebUIConfig.Configuration.Vault.TransitKeysListPath; path != "" {
			return path
		}
	}
	return "transit/keys"
}

// getTransitKeyCreateFormat returns the transit key create format from configuration
func getTransitKeyCreateFormat() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if format := factory.WebUIConfig.Configuration.Vault.TransitKeyCreateFmt; format != "" {
			return format
		}
	}
	return "transit/keys/%s"
}

// getExternalKeysListPath returns the external keys list path from configuration
func getExternalKeysListPath() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if path := factory.WebUIConfig.Configuration.Vault.KeyKVMetadataPath; path != "" {
			return path
		}
	}
	return "secret/metadata/k4keys"
}

// getTransitKeyRewrapFormat returns the transit key rewrap format from configuration
func getTransitKeyRewrapFormat() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if format := factory.WebUIConfig.Configuration.Vault.TransitKeyRewrapFmt; format != "" {
			return format
		}
	}
	return "transit/rewrap/%s"
}

// SyncKeyListen listens for key synchronization messages from Vault
func SyncKeyListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	logger.AppLog.Info("Vault key sync listener started")

	period := 5 * time.Minute
	if factory.WebUIConfig.Configuration.Vault != nil && factory.WebUIConfig.Configuration.Vault.SsmSync != nil && factory.WebUIConfig.Configuration.Vault.SsmSync.IntervalMinute > 0 {
		period = time.Duration(factory.WebUIConfig.Configuration.Vault.SsmSync.IntervalMinute) * time.Minute
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case msg := <-ssmSyncMsg:
			switch msg.Action {
			case "SYNC_OUR_KEYS":
				go syncOurKeys(msg.Action)
			case "SYNC_EXTERNAL_KEYS":
				go syncExternalKeys(msg.Action)
			case "SYNC_USERS":
				// Logic to synchronize users with Vault encryption user data that are not stored in Vault
				go SyncUsers()
			default:
				logger.AppLog.Warnf("Unknown SSM sync action: %s", msg.Action)
			}
			// Handle incoming SSM sync messages
		case <-ticker.C:
			// Periodic synchronization logic
			if err := VaultSyncInitDefault(ssmSyncMsg); err != nil {
				logger.AppLog.Errorf("VaultSyncInitDefault failed: %v", err)
			}
		}
	}
}

// VaultSyncInitDefault performs initial synchronization with Vault
func VaultSyncInitDefault(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	if readStopCondition() {
		logger.AppLog.Warn("Vault is down or has a problem, check if the component is running")
		return errors.New("vault is down")
	}

	logger.AppLog.Info("Starting default Vault synchronization")

	// Authenticate to Vault
	_, err := apiclient.LoginVault()
	if err != nil {
		logger.AppLog.Errorf("Failed to authenticate to Vault: %v", err)
		setStopCondition(true)
		return err
	}

	// Reset stop condition on successful authentication
	setStopCondition(false)

	// Enqueue default sync actions (mirror SSM behavior)
	ssmSyncMsg <- &ssm.SsmSyncMessage{Action: "SYNC_OUR_KEYS", Info: "Initial sync of internal keys"}
	ssmSyncMsg <- &ssm.SsmSyncMessage{Action: "SYNC_EXTERNAL_KEYS", Info: "Initial sync of external keys"}
	ssmSyncMsg <- &ssm.SsmSyncMessage{Action: "SYNC_USERS", Info: "Initial sync of users"}

	logger.AppLog.Info("Vault synchronization completed successfully")
	return nil
}

// HealthCheckVault performs a health check on the Vault connection
func HealthCheckVault() {
	logger.AppLog.Info("Performing Vault health check")

	// Ticker for periodic health checks (every 30 seconds)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		client, err := apiclient.GetVaultClient()
		if err != nil {
			logger.AppLog.Errorf("Vault health check failed - cannot get client: %v", err)
			setStopCondition(true)
			continue
		}

		// Check Vault health endpoint
		health, err := client.Sys().Health()
		if err != nil {
			logger.AppLog.Errorf("Vault health check failed: %v", err)
			setStopCondition(true)
			continue
		}

		if !health.Initialized {
			logger.AppLog.Warn("Vault is not initialized")
			setStopCondition(true)
			continue
		}

		if health.Sealed {
			logger.AppLog.Warn("Vault is sealed")
			setStopCondition(true)
			continue
		}

		logger.AppLog.Debugf("Vault health check passed - Version: %s, Cluster: %s", health.Version, health.ClusterName)
		setStopCondition(false)
	}
}

// readStopCondition safely reads the stop condition flag
func readStopCondition() bool {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	return StopVaultSyncFunction
}

// setStopCondition safely sets the stop condition flag
func setStopCondition(stop bool) {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	StopVaultSyncFunction = stop
	if stop {
		logger.AppLog.Warn("Vault sync function stopped")
	} else {
		logger.AppLog.Info("Vault sync function resumed")
	}
}
