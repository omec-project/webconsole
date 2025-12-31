package ssmsync

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
)

func SsmSyncInitDefault(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	// Initialize default SSM synchronization messages
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return
	}
	SyncKeys(ssm_constants.LABEL_ENCRYPTION_KEY, "SYNC_OUR_KEYS")
	for _, keyLabel := range ssm_constants.KeyLabelsInternalAllow {
		SyncKeys(keyLabel, "SYNC_OUR_KEYS")
	}

	ssmSyncMsg <- &ssm.SsmSyncMessage{Action: "SYNC_EXTERNAL_KEYS", Info: "Initial sync of keys"}
	ssmSyncMsg <- &ssm.SsmSyncMessage{Action: "SYNC_USERS", Info: "Initial sync of users"}
}

// Function that will be called concurrently to handle SSM synchronization
func SyncKeys(keyLabel, action string) {
	// Logic to synchronize keys with SSM

	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return
	}

	// channels
	k4listChanMDB := make(chan []configmodels.K4)
	k4listChanSSM := make(chan []ssm_models.DataKeyInfo)

	// First get the keys using a filter on keyLabel (mongodb query)
	go GetMongoDBLabelFilter(keyLabel, k4listChanMDB)

	// Then get the keys from SSM using the same keyLabel
	go getSSMLabelFilter(keyLabel, k4listChanSSM)

	// get the keys from both sources
	k4ListMDB := <-k4listChanMDB
	k4ListSSM := <-k4listChanSSM

	if k4ListMDB == nil || k4ListSSM == nil {
		ErrorSyncChan <- errors.New("invalid operation in ssm sync check the logs to read more information")
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

	// Case 1: Keys missing in both - create keys in the ssm and store in both MDB and SSM
	if len(mdbKeysMap) == 0 && len(ssmKeysMap) == 0 {
		// Create new key
		if action == "SYNC_OUR_KEYS" {
			logger.AppLog.Infof("No keys found in both MongoDB and SSM for label %s - creating new keys", keyLabel)
			for i := 0; i < factory.WebUIConfig.Configuration.SSM.SsmSync.MaxKeysCreate; i++ {
				go func() {
					newK4, err := createNewKeySSM(keyLabel, int32(i+1))
					if err != nil {
						logger.AppLog.Errorf("Failed to create new K4 key with label %s: %v", keyLabel, err)
					} else {
						// Store in MongoDB
						if err := StoreInMongoDB(newK4, keyLabel); err != nil {
							logger.AppLog.Errorf("Failed to store new K4 key in MongoDB: %v", err)
						}
					}
				}()
			}
		} else {
			logger.AppLog.Infof("No keys found in both MongoDB and SSM for label %s - skipping key creation as action is %s", keyLabel, action)
		}
	}

	// Case 2: Keys in MDB but not in SSM - delete to MongoDB
	for identifier, mdbKey := range mdbKeysMap {
		if _, existsInSSM := ssmKeysMap[identifier]; !existsInSSM {
			go func() {
				logger.AppLog.Infof("Key identifier %d exists in MDB but not in SSM - deleting to MongoDB", identifier)
				if err := DeleteKeyMongoDB(mdbKey); err != nil {
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
			if factory.WebUIConfig.Configuration.SSM.SsmSync.DeleteMissing {
				go func() {
					logger.AppLog.Infof("Removing key identifier %d from SSM as per policy", identifier)
					dataInfo := ssmKeysMap[identifier]
					k4 := configmodels.K4{
						K4_SNO:   byte(dataInfo.Id),
						K4_Label: keyLabel,
					}
					if err := deleteKeyToSSM(k4); err != nil {
						logger.AppLog.Errorf("Failed to remove key identifier %d from SSM: %v", identifier, err)
					} else {
						logger.AppLog.Infof("Successfully removed key identifier %d from SSM", identifier)
					}
				}()
			}
		}
	}

	// if not execute any cases (1,2,3), we assume keys are in sync and this is the case 4

	logger.AppLog.Infof("K4 key synchronization completed for label: %s", keyLabel)
}

func SyncUsers() {
	SyncUserMutex.Lock()
	defer SyncUserMutex.Unlock()

	coreUserSync()
}

func coreUserSync() {
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return
	}
	userList := GetUsersMDB()

	for _, user := range userList {
		// Logic to synchronize each user
		logger.AppLog.Infof("Synchronizing user: %s", user.UeId)
		// Add synchronization logic here
		go func() {
			subsData, err := GetSubscriberData(user.UeId)
			if err != nil {
				logger.AppLog.Errorf("Failed to get subscriber data for user %s: %v", user.UeId, err)
				return
			}
			if subsData == nil {
				logger.AppLog.Warnf("No subscriber data found for user %s", user.UeId)
				return
			}

			if subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm == 0 &&
				subsData.AuthenticationSubscription.K4_SNO == 0 {
				logger.AppLog.Warnf("User %s has no encryption key assigned we create a new one", user.UeId)
				// now we encrypt the key and store it back
				if factory.WebUIConfig.Configuration.SSM.IsEncryptAESGCM {
					encryptDataAESGCM(subsData, user)
				} else if factory.WebUIConfig.Configuration.SSM.IsEncryptAESCBC {
					encryptDataAESCBC(subsData, user)
				}
			}
		}()
	}
}

func encryptDataAESCBC(subsData *configmodels.SubsData, user configmodels.SubsListIE) {
	encryptRequest := ssm_models.EncryptRequest{
		KeyLabel:            ssm_constants.LABEL_ENCRYPTION_KEY_AES256,
		Plain:               subsData.AuthenticationSubscription.PermanentKey.PermanentKeyValue,
		EncryptionAlgorithm: ssm_constants.ALGORITHM_AES256_OurUsers,
	}

	apiClient := apiclient.GetSSMAPIClient()
	resp, r, err := apiClient.EncryptionAPI.EncryptData(apiclient.AuthContext).EncryptRequest(encryptRequest).Execute()
	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateAESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return
	}
	newSubAuthData := subsData.AuthenticationSubscription

	if resp.Cipher != "" {
		newSubAuthData.PermanentKey.PermanentKeyValue = resp.Cipher
		newSubAuthData.PermanentKey.EncryptionAlgorithm = ssm_constants.ALGORITHM_AES256_OurUsers
		newSubAuthData.K4_SNO = byte(resp.Id)
	}
	if resp.Iv != "" {
		newSubAuthData.PermanentKey.IV = resp.Iv
	}

	// now we store the new data do a update in mongoDB store
	err = configapi.SubscriberAuthenticationDataUpdate(user.UeId, &newSubAuthData)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber %s: %v", user.UeId, err)
		return
	}
	logger.WebUILog.Infof("Subscriber %s updated successfully", user.UeId)
}

func encryptDataAESGCM(subsData *configmodels.SubsData, user configmodels.SubsListIE) {
	aad := fmt.Sprintf("%s-%d-%d", subsData.UeId, subsData.AuthenticationSubscription.K4_SNO, subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm)
	aadBytes := []byte(aad) // Convertir a bytes

	encryptRequest := ssm_models.EncryptAESGCMRequest{
		KeyLabel: ssm_constants.LABEL_ENCRYPTION_KEY_AES256,
		Plain:    subsData.AuthenticationSubscription.PermanentKey.PermanentKeyValue,
		Aad:      hex.EncodeToString(aadBytes), // Codificar a hex
	}

	apiClient := apiclient.GetSSMAPIClient()
	resp, r, err := apiClient.EncryptionAPI.EncryptDataAESGCM(apiclient.AuthContext).EncryptAESGCMRequest(encryptRequest).Execute()
	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateAESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return
	}
	newSubAuthData := subsData.AuthenticationSubscription

	if resp.Cipher != "" {
		newSubAuthData.PermanentKey.PermanentKeyValue = resp.Cipher
		newSubAuthData.PermanentKey.EncryptionAlgorithm = ssm_constants.ALGORITHM_AES256_OurUsers
		newSubAuthData.K4_SNO = byte(resp.Id)
	}
	if resp.Iv != "" {
		newSubAuthData.PermanentKey.IV = resp.Iv
	}
	if resp.Tag != "" {
		newSubAuthData.PermanentKey.Tag = resp.Tag
	}
	newSubAuthData.PermanentKey.Aad = encryptRequest.Aad

	// now we store the new data do a update in mongoDB store
	err = configapi.SubscriberAuthenticationDataUpdate(user.UeId, &newSubAuthData)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber %s: %v", user.UeId, err)
		return
	}
	logger.WebUILog.Infof("Subscriber %s updated successfully", user.UeId)
}
