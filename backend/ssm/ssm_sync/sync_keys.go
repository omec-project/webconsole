package ssmsync

import (
	"sync"
	"time"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
)

var (
	SyncOurKeysMutex      sync.Mutex
	SyncExternalKeysMutex sync.Mutex
	SyncUserMutex         sync.Mutex
)

func SyncKeyListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	// Check if we need to stop the sync function before initializing
	if StopSSMsyncFunction {
		return
	}

	period := time.Duration(factory.WebUIConfig.Configuration.SSM.SsmSync.IntervalMinute) * time.Minute
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for !StopSSMsyncFunction {
		select {
		case msg := <-ssmSyncMsg:
			switch msg.Action {
			case "SYNC_OUR_KEYS":
				go syncOurKeys(msg.Action)
			case "SYNC_EXTERNAL_KEYS":
				go syncExternalKeys(msg.Action)
			case "SYNC_USERS":
				// Logic to synchronize users with SSM encryption user data that are not stored in SSM
				go SyncUsers()
			default:
				logger.AppLog.Warnf("Unknown SSM sync action: %s", msg.Action)
			}
			// Handle incoming SSM sync messages
		case <-ticker.C:
			// Periodic synchronization logic
			SsmSyncInitDefault(ssmSyncMsg)
		}
	}
}

func syncOurKeys(action string) {
	SyncOurKeysMutex.Lock()
	defer SyncOurKeysMutex.Unlock()

	// wait group
	var wg sync.WaitGroup

	// Logic to synchronize our keys with SSM this process check if we have keys like as AES, DES or DES3
	SyncKeys(ssm_constants.LABEL_ENCRYPTION_KEY, action)
	for _, keyLabel := range ssm_constants.KeyLabelsInternalAllow {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SyncKeys(keyLabel, action)
		}()
	}
	wg.Wait()
}

func syncExternalKeys(action string) {
	SyncExternalKeysMutex.Lock()
	defer SyncExternalKeysMutex.Unlock()
	// wait group
	var wg sync.WaitGroup

	// Logic to synchronize keys with SSM
	for _, keyLabel := range ssm_constants.KeyLabelsExternalAllow {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SyncKeys(keyLabel, action)
		}()
	}
	wg.Wait()
}
