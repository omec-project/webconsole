package ssmsync

import (
	"encoding/json"
	"fmt"
	"time"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

func readStopCondition() bool {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	return StopSSMsyncFunction
}

// Functions for SSM operations

func getSSMLabelFilter(keyLabel string, dataKeyInfoListChan chan []ssm_models.DataKeyInfo) {
	// Logic to get keys from SSM based on keyLabel

	logger.AppLog.Debugf("key label: %s", keyLabel)
	var getDataKeysRequest ssm_models.GetDataKeysRequest = ssm_models.GetDataKeysRequest{
		KeyLabel: keyLabel,
	}
	logger.AppLog.Debugf("Fetching keys from SSM with label: %s", getDataKeysRequest.KeyLabel)

	apiClient := apiclient.GetSSMAPIClient()

	resp, r, err := apiClient.KeyManagementAPI.GetDataKeys(apiclient.AuthContext).GetDataKeysRequest(getDataKeysRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GetDataKeys`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		dataKeyInfoListChan <- nil
		ErrorSyncChan <- err
		return
	}

	dataKeyInfoListChan <- resp.Keys
}

func deleteKeyToSSM(k4 configmodels.K4) error {
	logger.AppLog.Infof("Deleting key SNO %d with label %s from SSM", k4.K4_SNO, k4.K4_Label)

	apiClient := apiclient.GetSSMAPIClient()
	var deleteDataKeyRequest ssm_models.DeleteKeyRequest = ssm_models.DeleteKeyRequest{
		Id:       int32(k4.K4_SNO),
		KeyLabel: k4.K4_Label,
	}

	_, r, err := apiClient.KeyManagementAPI.DeleteKey(apiclient.AuthContext).DeleteKeyRequest(deleteDataKeyRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.DeleteKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return err
	}

	return nil
}

func createNewKeySSM(keyLabel string, id int32) (configmodels.K4, error) {
	var creator CreateKeySSM

	// Determine which creator to use based on key type embedded in label
	// Assuming labels follow pattern: K4_AES, K4_DES, K4_DES3
	switch keyLabel {
	case ssm_constants.LABEL_ENCRYPTION_KEY_AES128:
		creator = &CreateAES128SSM{}
	case ssm_constants.LABEL_ENCRYPTION_KEY_AES256:
		creator = &CreateAES256SSM{}
	case ssm_constants.LABEL_ENCRYPTION_KEY_DES3:
		creator = &CreateDes3SSM{}
	case ssm_constants.LABEL_ENCRYPTION_KEY_DES:
		creator = &CreateDesSSM{}
	default:
		return configmodels.K4{}, fmt.Errorf("unsupported key label: %s", keyLabel)
	}
	k4, err := creator.CreateNewKeySSM(keyLabel, id)

	k4.TimeCreated = time.Now()
	k4.TimeUpdated = k4.TimeCreated
	return k4, err
}

// Functions for MongoDB operations

func GetMongoDBLabelFilter(keyLabel string, k4listChan chan []configmodels.K4) {
	k4List := make([]configmodels.K4, 0)
	k4DataList, errGetMany := dbadapter.AuthDBClient.RestfulAPIGetMany(configapi.K4KeysColl, bson.M{"key_label": keyLabel})
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve k4 keys list with error: %+v", errGetMany)
		k4listChan <- nil
		ErrorSyncChan <- errGetMany
		return
	}
	if len(k4DataList) == 0 {
		k4listChan <- k4List
		return
	}

	var k4Data configmodels.K4
	for _, k4DataInterface := range k4DataList {
		err := json.Unmarshal(configmodels.MapToByte(k4DataInterface), &k4Data)
		if err != nil {
			k4listChan <- nil
			ErrorSyncChan <- err
			return
		}

		k4List = append(k4List, k4Data)
	}
	k4listChan <- k4List
}

func GetMongoDBAllK4(k4listChan chan []configmodels.K4) {
	k4List := make([]configmodels.K4, 0)
	k4DataList, errGetMany := dbadapter.AuthDBClient.RestfulAPIGetMany(configapi.K4KeysColl, bson.M{})
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve k4 keys list with error: %+v", errGetMany)
		k4listChan <- nil
		ErrorSyncChan <- errGetMany
		return
	}
	if len(k4DataList) == 0 {
		k4listChan <- k4List
		return
	}

	var k4Data configmodels.K4
	for _, k4DataInterface := range k4DataList {
		err := json.Unmarshal(configmodels.MapToByte(k4DataInterface), &k4Data)
		if err != nil {
			k4listChan <- nil
			ErrorSyncChan <- err
			return
		}

		k4List = append(k4List, k4Data)
	}
	k4listChan <- k4List
}

func StoreInMongoDB(k4 configmodels.K4, keyLabel string) error {
	logger.AppLog.Infof("Storing new key SNO %d in MongoDB with label %s", k4.K4_SNO, keyLabel)

	r, err := dbadapter.AuthDBClient.RestfulAPIGetOne(configapi.K4KeysColl, bson.M{"k4_sno": k4.K4_SNO, "key_label": keyLabel})

	if err != nil {
		logger.AppLog.Errorf("error: store K4 key in MongoDB %s", err)
		return err
	}
	if len(r) > 0 {
		logger.AppLog.Warn("K4 key in MongoDB exist")
		return err
	}

	k4Data := bson.M{
		"k4":           k4.K4,
		"k4_sno":       k4.K4_SNO,
		"key_label":    k4.K4_Label,
		"key_type":     k4.K4_Type,
		"time_created": time.Now(),
		"time_updated": time.Now(),
	}

	_, err = dbadapter.AuthDBClient.RestfulAPIPutOne(configapi.K4KeysColl, bson.M{"k4_sno": k4.K4_SNO, "key_label": keyLabel}, k4Data)
	if err != nil {
		logger.AppLog.Errorf("Failed to store K4 key in MongoDB: %v", err)
		return err
	}

	logger.AppLog.Infof("Successfully stored K4 key with SNO %d and label %s in MongoDB", k4.K4_SNO, keyLabel)
	return nil
}

func GetUsersMDB() []configmodels.SubsListIE {
	logger.WebUILog.Infoln("Get All Subscribers List")

	logger.WebUILog.Infoln("Get All Subscribers List")

	subsList := make([]configmodels.SubsListIE, 0)
	amDataList, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(configapi.AmDataColl, bson.M{})
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve subscribers list with error: %+v", errGetMany)
		return subsList
	}
	logger.AppLog.Infof("GetSubscribers: len: %d", len(amDataList))
	if len(amDataList) == 0 {
		return subsList
	}
	for _, amData := range amDataList {
		var subsData configmodels.SubsListIE

		err := json.Unmarshal(configmodels.MapToByte(amData), &subsData)
		if err != nil {
			logger.AppLog.Errorf("could not unmarshal subscriber %s", amData)
		}

		if servingPlmnId, plmnIdExists := amData["servingPlmnId"]; plmnIdExists {
			subsData.PlmnID = servingPlmnId.(string)
		}

		subsList = append(subsList, subsData)
	}

	return subsList
}

func GetSubscriberData(ueId string) (*configmodels.SubsData, error) {
	filterUeIdOnly := bson.M{"ueId": ueId}

	var subsData configmodels.SubsData

	authSubsDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(configapi.AuthSubsDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch authentication subscription data from DB: %+v", err)
		return &subsData, fmt.Errorf("failed to fetch authentication subscription data: %w", err)
	} // If all fetched data is empty, return error

	var authSubsData models.AuthenticationSubscription
	if authSubsDataInterface == nil {
		logger.WebUILog.Errorf("subscriber with ID %s not found", ueId)
		return &subsData, fmt.Errorf("subscriber with ID %s not found", ueId)
	} else {
		err := json.Unmarshal(configmodels.MapToByte(authSubsDataInterface), &authSubsData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling authentication subscription data: %+v", err)
			return &subsData, fmt.Errorf("failed to unmarshal authentication subscription data: %w", err)
		}
	}

	subsData = configmodels.SubsData{
		UeId:                       ueId,
		AuthenticationSubscription: authSubsData,
	}

	return &subsData, nil
}

func GetAllSubscriberData() ([]configmodels.SubsData, error) {
	filter := bson.M{}

	authSubsDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetMany(configapi.AuthSubsDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch authentication subscription data from DB: %+v", err)
		return nil, fmt.Errorf("failed to fetch authentication subscription data: %w", err)
	} // If all fetched data is empty, return error

	var subsDatas []configmodels.SubsData
	if authSubsDataInterface == nil {
		logger.WebUILog.Error("subscribers not found")
		return nil, fmt.Errorf("subscribers not found")
	} else {
		for _, authdata := range authSubsDataInterface {
			var authSubsData models.AuthenticationSubscription
			err := json.Unmarshal(configmodels.MapToByte(authdata), &authSubsData)
			if err != nil {
				logger.WebUILog.Errorf("error unmarshalling authentication subscription data: %+v", err)
				return nil, fmt.Errorf("failed to unmarshal authentication subscription data: %w", err)
			}
			subData := configmodels.SubsData{
				UeId:                       authdata["ueId"].(string),
				AuthenticationSubscription: authSubsData}
			subsDatas = append(subsDatas, subData)
		}
	}

	return subsDatas, nil
}

func DeleteKeyMongoDB(k4 configmodels.K4) error {
	logger.AppLog.Infof("Deleting key SNO %d with label %s from MongoDB", k4.K4_SNO, k4.K4_Label)

	err := dbadapter.AuthDBClient.RestfulAPIDeleteOne(configapi.K4KeysColl, bson.M{"k4_sno": k4.K4_SNO, "key_label": k4.K4_Label})
	return err
}
