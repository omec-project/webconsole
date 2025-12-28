package ssmsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

var CheckMutex, RotationMutex sync.Mutex

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
			// TODO: implement the check function that return a report about the key life
			CheckKeyHealth(ssmSyncMsg)

		case <-ticker90d.C:
			logger.AppLog.Info("Performing 90-day key rotation")
			// TODO: do the function to do the rotation for each key that grown 90 days living
			rotateExpiredKeys(ssmSyncMsg)
		}
	}
}

func CheckKeyHealth(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	// check the key life periodicly
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return errors.New("SSM is down")
	}
	// first sync the keys
	SsmSyncInitDefault(ssmSyncMsg)

	// now we get all keys in mongodb
	//channels
	k4listChanMDB := make(chan []configmodels.K4)

	// First get the keys using a filter on keyLabel (mongodb query)
	go GetMongoDBAllK4(k4listChanMDB)

	k4List := <-k4listChanMDB

	if k4List == nil {
		ErrorSyncChan <- errors.New("invalid operation in ssm sync check the logs to read more information")
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

	return nil
}

func rotateExpiredKeys(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	// rotate the keys that are older than 90 days
	if readStopCondition() {
		logger.AppLog.Warn("The ssm is down or have a problem check if that component is running")
		return errors.New("SSM DOWN")
	}
	// 1st syncronize the keys
	SsmSyncInitDefault(ssmSyncMsg)

	// 2nd get all keys filter by label and date
	k4listChanMDB := make(chan []configmodels.K4)
	go GetMongoDBAllK4(k4listChanMDB)
	k4List := <-k4listChanMDB

	if k4List == nil {
		ErrorSyncChan <- errors.New("invalid operation in ssm sync check the logs to read more information")
		return errors.New("invalid operation in ssm sync check the logs to read more information")
	}

	// Filter keys older than 90 days
	now := time.Now()
	var expiredKeys []configmodels.K4

	for _, k4 := range k4List {
		daysSinceCreation := int(now.Sub(k4.TimeCreated).Hours() / 24)
		if daysSinceCreation >= 90 {
			expiredKeys = append(expiredKeys, k4)
		}
	}

	logger.AppLog.Infof("Found %d expired keys (≥90 days old) to rotate", len(expiredKeys))

	if len(expiredKeys) == 0 {
		logger.AppLog.Info("No expired keys found. Rotation complete.")
		return nil
	}

	// the next steps are integrated in rotateKey function
	// 3rd get the users that use this key use a concurrent algoritm
	// 4th decrypt the ki for the user
	// 5th delete the old key in HSM and mongoDB
	// 6th generate a same key type use the same id and key label
	// 7th encrypt the ki with the new secret key
	// 8th save the datas (save the new cipher ki and the new k4 if is necessary)
	for _, k4exp := range expiredKeys {
		go rotateKey(k4exp)
	}

	logger.AppLog.Infof("Key rotation process initiated for %d keys", len(expiredKeys))

	return nil
}

func rotateKey(k4 configmodels.K4) {
	// Get users associated with the key to be rotated
	userToRotateKi, err := getUsersForRotation(k4)
	if err != nil {
		logger.AppLog.Errorf("failed to get users for rotation: %v", err)
		return
	}
	if len(userToRotateKi) == 0 {
		logger.AppLog.Infof("No users found for key rotation for K4_SNO: %d, Label: %s", k4.K4_SNO, k4.K4_Label)
		return
	}

	// Proceed with key rotation
	// Decrypt the KI for each user before deleting the key. Match the results with users.
	var wg sync.WaitGroup
	for _, user := range userToRotateKi {
		wg.Add(1)
		go func(user models.AuthenticationSubscription) {
			defer wg.Done()
			// operate on the slice element address so decrypted KI is stored back into the slice
			decryptUserKI(&user, k4)
		}(user)
	}
	wg.Wait()

	// In this point all users have their KI decrypted and stored in userToRotateKi slice

	//Delete the key for the HSM and create a new one with the same key label and k4_sno
	logger.AppLog.Infof("Rotating key K4_SNO: %d, Label: %s", k4.K4_SNO, k4.K4_Label)
	if err := deleteKeyToSSM(k4); err != nil {
		logger.AppLog.Errorf("failed to delete old key: %v", err)
		return
	}

	newK4, err := createNewKeySSM(k4.K4_Label, int32(k4.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to create new key: %v", err)
		return
	}
	// Proceed with key encryption for each user (use WaitGroup to wait for all encryptions)
	var wgEnc sync.WaitGroup
	for ueId, user := range userToRotateKi {
		wgEnc.Add(1)
		go func(u models.AuthenticationSubscription, id string) {
			defer wgEnc.Done()
			encryptUserKey(&u, newK4, id)
		}(user, ueId)
	}
	wgEnc.Wait()
}

func decryptUserKI(user *models.AuthenticationSubscription, k4 configmodels.K4) {
	// 1. Configure the SSM client
	ssmClient := apiclient.GetSSMAPIClient()

	// 2. Prepare the decryption request
	encryptionAlgorithm := int(user.PermanentKey.EncryptionAlgorithm)
	keyLabel := k4.K4_Label
	keyId := k4.K4_SNO
	encryptedKiHex := user.PermanentKey.PermanentKeyValue

	decryptReq := ssm_models.DecryptRequest{
		KeyLabel:            keyLabel,
		Cipher:              encryptedKiHex,
		EncryptionAlgorithm: int32(encryptionAlgorithm),
		Id:                  int32(keyId),
		Iv:                  user.PermanentKey.IV,
	}

	// 3. Execute the SSM API call
	decryptedResp, _, decryptErr := ssmClient.EncryptionAPI.DecryptData(apiclient.AuthContext).DecryptRequest(decryptReq).Execute()
	if decryptErr != nil {
		logger.AppLog.Errorf("SSM decryption failed: %+v", decryptErr)
		return
	}

	// 4. Process the SSM response
	// The SSM response 'Plain' is in hexadecimal format.
	user.PermanentKey.PermanentKeyValue = decryptedResp.Plain
}

func encryptUserKey(user *models.AuthenticationSubscription, k4 configmodels.K4, ueId string) {
	// now we encrypt the key and store it back
	var encryptRequest ssm_models.EncryptRequest = ssm_models.EncryptRequest{
		KeyLabel:            k4.K4_Label,
		Plain:               user.PermanentKey.PermanentKeyValue,
		EncryptionAlgorithm: int32(ssm_constants.LabelAlgorithmMap[k4.K4_Label]),
	}

	apiClient := apiclient.GetSSMAPIClient()

	resp, r, err := apiClient.EncryptionAPI.EncryptData(apiclient.AuthContext).EncryptRequest(encryptRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateAESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
	}

	if resp.Cipher != "" {
		user.PermanentKey.PermanentKeyValue = resp.Cipher
		user.PermanentKey.EncryptionAlgorithm = ssm_constants.ALGORITHM_AES256_OurUsers
		user.K4_SNO = byte(resp.Id)
	}
	if resp.Iv != "" {
		user.PermanentKey.IV = resp.Iv
	}

	// now we store the new data do a update in mongoDB store
	err = configapi.SubscriberAuthenticationDataUpdate(ueId, user)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber %s: %v", ueId, err)
		return
	}
	logger.WebUILog.Infof("Subscriber %s updated successfully", ueId)

	// msg := configmodels.ConfigMessage{
	// 	MsgType:     configmodels.Sub_data,
	// 	MsgMethod:   configmodels.Put_op,
	// 	AuthSubData: user,
	// 	Imsi:        ueId,
	// }
	// cfgChannel <- &msg
}

func getUsersForRotation(k4 configmodels.K4) (map[string]models.AuthenticationSubscription, error) {
	authSubList := make(map[string]models.AuthenticationSubscription)
	authDataList, errGetMany := dbadapter.AuthDBClient.RestfulAPIGetMany(configapi.AuthSubsDataColl,
		bson.M{
			"k4_sno":                           int(k4.K4_SNO),
			"permanentKey.encryptionAlgorithm": ssm_constants.LabelAlgorithmMap[k4.K4_Label],
		})
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve k4 keys list with error: %+v", errGetMany)
	}

	for _, authSub := range authDataList {
		var authSubsData models.AuthenticationSubscription
		if authSub != nil {
			err := json.Unmarshal(configmodels.MapToByte(authSub), &authSubsData)
			if err != nil {
				logger.WebUILog.Errorf("error unmarshalling authentication subscription data: %+v", err)
				return nil, fmt.Errorf("failed to unmarshal authentication subscription data: %w", err)
			}
			authSubList[authSub["ueId"].(string)] = authSubsData
		}
	}
	return authSubList, nil
}
