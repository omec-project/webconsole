package vaultsync

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
)

var ssmSyncMessage chan *ssm.SsmSyncMessage

func SetSyncChanHandle(ch chan *ssm.SsmSyncMessage) {
	ssmSyncMessage = ch
}

func handleSyncKey(c *gin.Context) {
	logger.AppLog.Debug("Init handle sync key")

	// Try to acquire locks without blocking - if any is already held, return busy
	if !SyncOurKeysMutex.TryLock() {
		logger.AppLog.Warn("SyncOurKeysMutex is already held, sync in progress")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "sync for internal keys already in progress"})
		return
	}
	defer SyncOurKeysMutex.Unlock()

	if !SyncExternalKeysMutex.TryLock() {
		logger.AppLog.Warn("SyncExternalKeysMutex is already held, sync in progress")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "sync for external keys already in progress"})
		return
	}
	defer SyncExternalKeysMutex.Unlock()

	if !SyncUserMutex.TryLock() {
		logger.AppLog.Warn("SyncUserMutex is already held, sync in progress")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "sync for users already in progress"})
		return
	}
	defer SyncUserMutex.Unlock()

	logger.AppLog.Debug("All locks acquired, starting sync operations")

	// Authenticate to Vault
	_, err := apiclient.LoginVault()
	if err != nil {
		logger.AppLog.Errorf("Failed to authenticate to Vault: %v", err)
		return
	}

	// Logic to synchronize our keys with Vault - this process checks if we have keys like AES
	logger.AppLog.Debugf("Starting sync for internal keys with label: %s", ssm_constants.LABEL_ENCRYPTION_KEY_AES256)
	syncOurKeys("SYNC_OUR_KEYS")
	logger.AppLog.Debug("Internal keys sync completed")

	// Logic to synchronize external keys with Vault
	logger.AppLog.Debugf("Starting sync for %d external key labels", len(ssm_constants.KeyLabelsExternalAllow))
	syncExternalKeysInternal("SYNC_EXTERNAL_KEYS")
	logger.AppLog.Debug("All external keys synced")

	// Synchronize users
	logger.AppLog.Debug("Starting core vault user sync")
	coreVaultUserSync()
	logger.AppLog.Debug("Core vault user sync completed")

	c.JSON(http.StatusOK, gin.H{"success": "sync function ran successfully"})
	logger.AppLog.Debug("Sync key handler finished successfully")
}

func handleCheckK4Life(c *gin.Context) {
	// Try to acquire all locks individually
	logger.AppLog.Debug("Init handle check k4 life")
	checkLocked := CheckMutex.TryLock()
	rotationLocked := RotationMutex.TryLock()

	// If any lock failed, cleanup and return error
	if !checkLocked || !rotationLocked {
		// Unlock only the ones we successfully locked
		if checkLocked {
			CheckMutex.Unlock()
		}
		if rotationLocked {
			RotationMutex.Unlock()
		}

		c.JSON(http.StatusTooManyRequests, gin.H{"error": "the operation check life k4 or rotation k4 is running"})
		return
	}

	defer CheckMutex.Unlock()
	defer RotationMutex.Unlock()
	if err := checkKeyHealth(ssmSyncMessage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Vault check-k4-life not implemented"})
}

func handleRotationKey(c *gin.Context) {
	// Try to acquire all locks individually
	logger.AppLog.Debug("Init handle rotation key")

	checkLocked := CheckMutex.TryLock()
	rotationLocked := RotationMutex.TryLock()

	// If any lock failed, cleanup and return error
	if !checkLocked || !rotationLocked {
		// Unlock only the ones we successfully locked
		if checkLocked {
			CheckMutex.Unlock()
		}
		if rotationLocked {
			RotationMutex.Unlock()
		}

		c.JSON(http.StatusTooManyRequests, gin.H{"error": "the operation check life k4 or rotation k4 is running"})
		return
	}

	defer CheckMutex.Unlock()
	defer RotationMutex.Unlock()

	if err := rotateInternalTransitKey(internalKeyLabel, ssmSyncMessage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Vault internal key rotation triggered"})
}
