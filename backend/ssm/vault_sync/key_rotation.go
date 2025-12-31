package vaultsync

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	ssmsync "github.com/omec-project/webconsole/backend/ssm/ssm_sync"
	"github.com/omec-project/webconsole/configmodels"
)

var CheckMutex, RotationMutex sync.Mutex

// KeyRotationListen handles rotation events for the internal transit key
func KeyRotationListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	ticker24h := time.NewTicker(24 * time.Hour)
	ticker90d := time.NewTicker(90 * 24 * time.Hour)
	defer ticker24h.Stop()
	defer ticker90d.Stop()

	logger.AppLog.Info("Key rotation listener started")

	for {
		select {
		case <-ticker24h.C:
			logger.AppLog.Info("Performing daily key health check")
			if err := checkKeyHealth(ssmSyncMsg); err != nil {
				logger.AppLog.Errorf("Error during key health check: %v", err)
			}

		case <-ticker90d.C:
			logger.AppLog.Info("Performing 90-day key rotation")
			if err := rotateInternalTransitKey(internalKeyLabel, ssmSyncMsg); err != nil {
				logger.AppLog.Errorf("Error rotating internal transit key: %v", err)
			}
		}
	}
}

func checkKeyHealth(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	// check the key life periodicly
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return errors.New("SSM is down")
	}
	// first sync the keys
	if err := VaultSyncInitDefault(ssmSyncMsg); err != nil {
		return err
	}

	// now we get all keys in mongodb
	// channels
	k4listChanMDB := make(chan []configmodels.K4)

	// First get the keys using a filter on keyLabel (mongodb query)
	go ssmsync.GetMongoDBAllK4(k4listChanMDB)

	k4List := <-k4listChanMDB

	if k4List == nil {
		ssmsync.ErrorSyncChan <- errors.New("invalid operation in ssm sync check the logs to read more information")
		return errors.New("invalid operation in ssm sync check the logs to read more information")
	}

	// Group keys by remaining days until 90-day expiration
	var firstHalf []configmodels.K4    // 45-90 days remaining
	var secondHalf []configmodels.K4   // 0-44 days remaining
	var criticalKeys []configmodels.K4 // 5 or fewer days remaining

	now := time.Now()

	for _, k4 := range k4List {
		// Calculate days since creation
		daysSinceCreation := int(now.Sub(k4.TimeCreated).Hours() / 24)
		daysRemaining := 90 - daysSinceCreation

		// Critical keys: 5 days or less to expiration
		if daysRemaining <= 5 && daysRemaining >= 0 {
			criticalKeys = append(criticalKeys, k4)
		}

		// Group into halves
		if daysRemaining >= 45 {
			firstHalf = append(firstHalf, k4)
		} else if daysRemaining >= 0 {
			secondHalf = append(secondHalf, k4)
		}
		// Keys with daysRemaining < 0 are already expired (not grouped)
	}

	// Print results
	logger.AppLog.Infof("=== Key Health Check Results ===")
	logger.AppLog.Infof("Total keys analyzed: %d", len(k4List))
	logger.AppLog.Infof("Keys with 45-90 days remaining: %d", len(firstHalf))
	logger.AppLog.Infof("Keys with 0-44 days remaining: %d", len(secondHalf))
	logger.AppLog.Infof("🚨 CRITICAL: Keys expiring in ≤5 days: %d", len(criticalKeys))

	// Log critical keys details
	if len(criticalKeys) > 0 {
		logger.AppLog.Warn("Critical keys requiring immediate attention:")
		for _, k4 := range criticalKeys {
			daysSinceCreation := int(now.Sub(k4.TimeCreated).Hours() / 24)
			daysRemaining := 90 - daysSinceCreation
			logger.AppLog.Warnf("  - K4_SNO: %d, Label: %s, Days remaining: %d", k4.K4_SNO, k4.K4_Label, daysRemaining)
		}
	}

	client, err := apiclient.GetVaultClient()
	if err != nil {
		return fmt.Errorf("get vault client: %w", err)
	}

	latest, err := getLatestTransitKeyVersion(client, internalKeyLabel, "opt2")
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	LatestKeyVersion = latest
	return nil
}

func rotateInternalTransitKey(keyLabel string, ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	if readStopCondition() {
		return errors.New("vault is down; skipping rotation")
	}

	if err := VaultSyncInitDefault(ssmSyncMsg); err != nil {
		return err
	}

	client, err := apiclient.GetVaultClient()
	if err != nil {
		return fmt.Errorf("get vault client: %w", err)
	}

	if apiclient.VaultAuthToken == "" {
		if _, err := apiclient.LoginVault(); err != nil {
			setStopCondition(true)
			return fmt.Errorf("authenticate vault: %w", err)
		}
	}

	rotateFmt := "transit/keys/%s/rotate"
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if f := factory.WebUIConfig.Configuration.Vault.TransitKeyRotateFmt; f != "" {
			rotateFmt = f
		}
	}
	rotatePath := fmt.Sprintf(rotateFmt, keyLabel)
	if _, err := client.Logical().Write(rotatePath, nil); err != nil {
		return fmt.Errorf("rotate transit key %s: %w", keyLabel, err)
	}
	LatestKeyVersion++
	return nil
}
