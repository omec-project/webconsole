package ssmsync

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
)

var ssmSyncMessage chan *ssm.SsmSyncMessage

func setSyncChanHandle(ch chan *ssm.SsmSyncMessage) {
	ssmSyncMessage = ch
}

func handleSyncKey(c *gin.Context) {
	// Try to get the priority
	logger.AppLog.Debug("Init handle sync key")

	externalLocked := SyncExternalKeysMutex.TryLock()
	ourKeysLocked := SyncOurKeysMutex.TryLock()
	userLocked := SyncUserMutex.TryLock()

	// If any lock failed, cleanup and return error
	if !externalLocked || !ourKeysLocked || !userLocked {
		// Unlock only the ones we successfully locked
		if externalLocked {
			SyncExternalKeysMutex.Unlock()
		}
		if ourKeysLocked {
			SyncOurKeysMutex.Unlock()
		}
		if userLocked {
			SyncUserMutex.Unlock()
		}

		c.JSON(http.StatusTooManyRequests, gin.H{"error": "sync function is running"})
		return
	}

	defer SyncExternalKeysMutex.Unlock()
	defer SyncOurKeysMutex.Unlock()
	defer SyncUserMutex.Unlock()

	// wait group
	var wg sync.WaitGroup

	// Logic to synchronize our keys with SSM this process check if we have keys like as AES, DES or DES3
	wg.Add(1)
	go func() {
		defer wg.Done()
		SyncKeys(ssm_constants.LABEL_ENCRYPTION_KEY, "SYNC_OUR_KEYS")
	}()
	for _, keyLabel := range ssm_constants.KeyLabelsInternalAllow {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SyncKeys(keyLabel, "SYNC_OUR_KEYS")
		}()
	}

	// Logic to synchronize keys with SSM
	for _, keyLabel := range ssm_constants.KeyLabelsExternalAllow {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SyncKeys(keyLabel, "SYNC_EXTERNAL_KEYS")
		}()
	}

	wg.Wait()

	coreUserSync()

	c.JSON(http.StatusOK, gin.H{"success": "sync function run successfully"})
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

	// Logic for the handle
	err := CheckKeyHealth(ssmSyncMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error: " + err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{"success": "sync function run successfully"})
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

	err := rotateExpiredKeys(ssmSyncMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error: " + err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{"success": "rotation function run successfully"})
}
