package configapi

import (
	"encoding/json"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"testing"
)

func Test_handleSubscriberPost5G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origImsiData := imsiData
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	origPostData := postData
	defer func() {
		subscriberAuthData = origSubscriberAuthData
		imsiData = origImsiData
		postData = origPostData
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
	subscriberAuthData = DatabaseSubscriberAuthenticationData{}
	configMsg := configmodels.ConfigMessage{
		AuthSubData: &models.AuthenticationSubscription{
			AuthenticationManagementField: "8000",
			AuthenticationMethod:          "5G_AKA",
			Milenage: &models.Milenage{
				Op: &models.Op{
					EncryptionAlgorithm: 0,
					EncryptionKey:       0,
				},
			},
			Opc: &models.Opc{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpcValue:            "8e27b6af0e692e750f32667a3b14605d",
			},
			PermanentKey: &models.PermanentKey{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
			},
			SequenceNumber: "16f3b3f70fc2",
		},
	}
	postData = make([]map[string]interface{}, 0)
	imsiData = make(map[string]*models.AuthenticationSubscription)
	dbadapter.AuthDBClient = &MockMongoPost{}
	dbadapter.CommonDBClient = &MockMongoPost{}
	postErr := handleSubscriberPost(ueId, configMsg.AuthSubData, subscriberAuthData)
	if postErr != nil {
		t.Errorf("Could not handle subscriber post: %v", postErr)
	}
	expectedAuthSubCollection := authSubsDataColl
	expectedAmDataCollection := amDataColl
	if postData[0]["coll"] != expectedAuthSubCollection {
		t.Errorf("Expected collection %v, got %v", expectedAuthSubCollection, postData[0]["coll"])
	}
	if postData[1]["coll"] != expectedAmDataCollection {
		t.Errorf("Expected collection %v, got %v", expectedAmDataCollection, postData[1]["coll"])
	}

	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(postData[0]["filter"], expectedFilter) {
		t.Errorf("Expected filter %v, got %v", expectedFilter, postData[0]["filter"])
	}
	if !reflect.DeepEqual(postData[1]["filter"], expectedFilter) {
		t.Errorf("Expected filter %v, got %v", expectedFilter, postData[1]["filter"])
	}

	var authSubResult models.AuthenticationSubscription
	result := postData[0]["data"].(map[string]interface{})
	err := json.Unmarshal(configmodels.MapToByte(result), &authSubResult)
	if err != nil {
		t.Errorf("Could not unmarshall result %v", result)
	}
	if !reflect.DeepEqual(configMsg.AuthSubData, &authSubResult) {
		t.Errorf("Expected authSubData %v, got %v", configMsg.AuthSubData, &authSubResult)
	}
	amDataResult := postData[1]["data"].(map[string]interface{})
	if amDataResult["ueId"] != ueId {
		t.Errorf("Expected ueId %v, got %v", ueId, amDataResult["ueId"])
	}
	if imsiData[ueId] != nil {
		t.Errorf("Expected no ueId in memory, got %v", imsiData[ueId])
	}
}

func Test_handleSubscriberPost4G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origImsiData := imsiData
	origCommonDBClient := dbadapter.CommonDBClient
	origPostData := postData
	defer func() {
		subscriberAuthData = origSubscriberAuthData
		imsiData = origImsiData
		postData = origPostData
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
	subscriberAuthData = MemorySubscriberAuthenticationData{}
	configMsg := configmodels.ConfigMessage{
		AuthSubData: &models.AuthenticationSubscription{
			AuthenticationManagementField: "8000",
			AuthenticationMethod:          "5G_AKA",
			Milenage: &models.Milenage{
				Op: &models.Op{
					EncryptionAlgorithm: 0,
					EncryptionKey:       0,
				},
			},
			Opc: &models.Opc{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpcValue:            "8e27b6af0e692e750f32667a3b14605d",
			},
			PermanentKey: &models.PermanentKey{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
			},
			SequenceNumber: "16f3b3f70fc2",
		},
	}

	postData = make([]map[string]interface{}, 0)
	imsiData = make(map[string]*models.AuthenticationSubscription)
	dbadapter.CommonDBClient = &MockMongoPost{}
	postErr := handleSubscriberPost(ueId, configMsg.AuthSubData, subscriberAuthData)
	if postErr != nil {
		t.Errorf("Could not handle subscriber post: %v", postErr)
	}

	expectedAmDataCollection := amDataColl
	if postData[0]["coll"] != expectedAmDataCollection {
		t.Errorf("Expected collection %v, got %v", expectedAmDataCollection, postData[0]["coll"])
	}

	expected_filter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
		t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
	}

	AmDataResult := postData[0]["data"].(map[string]interface{})
	if AmDataResult["ueId"] != ueId {
		t.Errorf("Expected ueId %v, got %v", ueId, AmDataResult["ueId"])
	}
	if !reflect.DeepEqual(imsiData[ueId], configMsg.AuthSubData) {
		t.Errorf("Expected authSubData %v in memory, got %v ", configMsg.AuthSubData, imsiData[ueId])
	}
}

func Test_handleSubscriberDelete5G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	origDeleteData := deleteData
	defer func() {
		subscriberAuthData = origSubscriberAuthData
		deleteData = origDeleteData
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
	subscriberAuthData = DatabaseSubscriberAuthenticationData{}

	deleteData = make([]map[string]interface{}, 0)
	dbadapter.AuthDBClient = &MockMongoDeleteOne{}
	dbadapter.CommonDBClient = &MockMongoDeleteOne{}
	delErr := handleSubscriberDelete(ueId, subscriberAuthData)
	if delErr != nil {
		t.Errorf("Could not handle subscriber delete: %v", delErr)
	}
	expectedAuthSubCollection := authSubsDataColl
	expectedAmDataCollection := amDataColl
	if deleteData[0]["coll"] != expectedAuthSubCollection {
		t.Errorf("Expected collection %v, got %v", expectedAuthSubCollection, deleteData[0]["coll"])
	}
	if deleteData[1]["coll"] != expectedAmDataCollection {
		t.Errorf("Expected collection %v, got %v", expectedAmDataCollection, deleteData[1]["coll"])
	}

	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(deleteData[0]["filter"], expectedFilter) {
		t.Errorf("Expected filter %v, got %v", expectedFilter, deleteData[0]["filter"])
	}
	if !reflect.DeepEqual(deleteData[1]["filter"], expectedFilter) {
		t.Errorf("Expected filter %v, got %v", expectedFilter, deleteData[1]["filter"])
	}
}

func Test_handleSubscriberDelete4G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origImsiData := imsiData
	origCommonDBClient := dbadapter.CommonDBClient
	origDeleteData := deleteData
	defer func() {
		subscriberAuthData = origSubscriberAuthData
		imsiData = origImsiData
		deleteData = origDeleteData
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
	subscriberAuthData = MemorySubscriberAuthenticationData{}

	deleteData = make([]map[string]interface{}, 0)
	imsiData = make(map[string]*models.AuthenticationSubscription)
	imsiData[ueId] = &models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            "8e27b6af0e692e750f32667a3b14605d",
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
		},
		SequenceNumber: "16f3b3f70fc2",
	}
	dbadapter.CommonDBClient = &MockMongoDeleteOne{}
	delErr := handleSubscriberDelete(ueId, subscriberAuthData)
	if delErr != nil {
		t.Errorf("Could not handle subscriber delete: %v", delErr)
	}

	expectedAmDataCollection := "subscriptionData.provisionedData.amData"
	if deleteData[0]["coll"] != expectedAmDataCollection {
		t.Errorf("Expected collection %v, got %v", expectedAmDataCollection, deleteData[0]["coll"])
	}

	expected_filter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(deleteData[0]["filter"], expected_filter) {
		t.Errorf("Expected filter %v, got %v", expected_filter, deleteData[0]["filter"])
	}

	if imsiData[ueId] != nil {
		t.Errorf("Expected no ueId in memory, got %v", imsiData[ueId])
	}
}
