package vaultsync

import (
	"errors"
	"strconv"
	"sync"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	ssm_models "github.com/networkgcorefullcode/ssm/models"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	ssmsync "github.com/omec-project/webconsole/backend/ssm/ssm_sync"
	"github.com/omec-project/webconsole/configmodels"
)

var SyncOurKeysMutex sync.Mutex
var SyncExternalKeysMutex sync.Mutex
var SyncUserMutex sync.Mutex

func syncOurKeys(action string) {
	SyncOurKeysMutex.Lock()
	defer SyncOurKeysMutex.Unlock()

	// Logic to synchronize our keys with SSM this process check if we have keys like as AES, DES or DES3
	// SyncKeys(ssm_constants.LABEL_ENCRYPTION_KEY, action)
	SyncKeys(ssm_constants.LABEL_ENCRYPTION_KEY_AES256, action)
}

func syncExternalKeys(action string) {
	SyncExternalKeysMutex.Lock()
	defer SyncExternalKeysMutex.Unlock()
	syncExternalKeysInternal(action)
}

// syncExternalKeysInternal performs external key sync without acquiring the mutex
// Use this when the mutex is already held by the caller
func syncExternalKeysInternal(action string) {
	// wait group
	var wg sync.WaitGroup

	// Logic to synchronize keys with SSM
	for _, keyLabel := range ssm_constants.KeyLabelsExternalAllow {
		wg.Add(1)
		go func(label string) {
			defer wg.Done()
			SyncKeys(label, action)
		}(keyLabel)
	}
	wg.Wait()
}

// syncOurKeys ensures our internal AES256-GCM key exists in Vault transit engine
func SyncKeys(keyLabel, action string) {

	// Logic to synchronize keys with SSM
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return
	}

	// Case 1: Actions is SYNC_OUR_KEYS
	if action == "SYNC_OUR_KEYS" {
		logger.AppLog.Info("Create the key that encript our subs datas")
		newK4, err := createNewKeyVaultTransit(keyLabel)
		if err != nil {
			logger.AppLog.Errorf("Failed to create new K4 key with label %s: %v", keyLabel, err)
		} else {
			// Store in MongoDB
			if err := ssmsync.StoreInMongoDB(newK4, keyLabel); err != nil {
				logger.AppLog.Errorf("Failed to store new K4 key in MongoDB: %v", err)
			}
		}
		return
	}

	//channels
	k4listChanMDB := make(chan []configmodels.K4)
	k4listChanSSM := make(chan []ssm_models.DataKeyInfo)

	// First get the keys using a filter on keyLabel (mongodb query)
	go ssmsync.GetMongoDBLabelFilter(keyLabel, k4listChanMDB)

	// Then get the keys from SSM using the same keyLabel
	go getVaultLabelFilter(keyLabel, k4listChanSSM)

	// get the keys from both sources
	k4ListMDB := <-k4listChanMDB
	k4ListSSM := <-k4listChanSSM

	if k4ListMDB == nil || k4ListSSM == nil {
		ssmsync.ErrorSyncChan <- errors.New("invalid operation in ssm sync check the logs to read more information")
		return
	}

	// now we can compare both lists and synchronize as needed
	// cases to handle:
	// 1. Keys missing in both -> create new keys and store in both MDB and SSM
	// 2. Keys in MDB but not in SSM -> delete to MongoDB
	// 3. Keys in SSM but not in MDB -> log warning or remove from SSM based on policy or store in MDB
	// 4. Keys in both and same -> no action needed

	logger.AppLog.Infof("Starting K4 key synchronization for label: %s", keyLabel)
	logger.AppLog.Debugf("Keys from MongoDB: %d, Keys from SSM: %d", len(k4ListMDB), len(k4ListSSM))

	// Create maps for efficient lookup
	mdbKeysMap := make(map[string]configmodels.K4)
	for _, k4 := range k4ListMDB {
		mdbKeysMap[strconv.Itoa(int(k4.K4_SNO))+keyLabel] = k4
	}

	ssmKeysMap := make(map[string]ssm_models.DataKeyInfo)
	for _, k4 := range k4ListSSM {
		// Assuming DataKeyInfo has a field for key ID/SNO
		ssmKeysMap[strconv.Itoa(int(k4.Id))+keyLabel] = k4
	}

	// Case 2: Keys in MDB but not in SSM - delete to MongoDB
	for identifier, mdbKey := range mdbKeysMap {
		if _, existsInSSM := ssmKeysMap[identifier]; !existsInSSM {
			go func() {
				logger.AppLog.Infof("Key identifier %d exists in MDB but not in SSM - deleting to MongoDB", identifier)
				if err := ssmsync.DeleteKeyMongoDB(mdbKey); err != nil {
					logger.AppLog.Errorf("Failed to delete key identifier %d from MongoDB: %v", identifier, err)
				} else {
					logger.AppLog.Infof("Successfully deleted key identifier %d from MongoDB", identifier)
				}
			}()
		}
	}

	// Case 3: Keys in SSM but not in MDB - log warning
	for identifier := range ssmKeysMap {
		if _, existsInMDB := mdbKeysMap[identifier]; !existsInMDB {
			logger.AppLog.Warnf("Key identifier %d exists in SSM but not in MongoDB - Label: %s", identifier, keyLabel)
			// Policy decision: we can either remove from SSM or just log
			// For safety, we'll just log by default
			// To remove from SSM, uncomment:
			if factory.WebUIConfig.Configuration.Vault.SsmSync.DeleteMissing {
				go func() {
					logger.AppLog.Infof("Removing key identifier %d from SSM as per policy", identifier)
					dataInfo := ssmKeysMap[identifier]
					k4 := configmodels.K4{
						K4_SNO:   byte(dataInfo.Id),
						K4_Label: keyLabel,
					}
					if err := deleteKeyToVault(k4); err != nil {
						logger.AppLog.Errorf("Failed to remove key identifier %d from SSM: %v", identifier, err)
					} else {
						logger.AppLog.Infof("Successfully removed key identifier %d from SSM", identifier)
					}
				}()
			}
		}
	}

}
