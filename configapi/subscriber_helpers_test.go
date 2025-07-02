// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type mockDB struct {
	getOneFunc    func(collName string, filter bson.M) (map[string]interface{}, error)
	deleteOneFunc func(collName string, filter bson.M) error
	postFunc      func(collName string, filter bson.M, postData map[string]interface{}) (bool, error)
	dbadapter.DBInterface
}

func (m *mockDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	return m.getOneFunc(collName, filter)
}

func (m *mockDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return m.postFunc(collName, filter, postData)
}

func (m *mockDB) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return m.deleteOneFunc(collName, filter)
}

func TestSubscriberAuthenticationDataCreate_Success(t *testing.T) {
	authCalled, commonCalled := false, false

	authDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
			authCalled = true
			return true, nil
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			t.Error("rollback should not be called on success")
			return nil
		},
	}

	commonDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
			commonCalled = true
			return true, nil
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			t.Error("should not be called")
			return nil
		},
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	sub := DatabaseSubscriberAuthenticationData{}
	subsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "c9e8763286b5b9ffbdf56e1297d0887b",
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            "981d464c7c52eb6e5036234984ad0bcf",
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "5122250214c33e723a5dd523fc145fc0",
		},
		SequenceNumber: "16f3b3f70fc2",
	}
	err := sub.SubscriberAuthenticationDataCreate("imsi-1", &subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !authCalled || !commonCalled {
		t.Errorf("expected both AuthDB and CommonDB to be called")
	}
}

func TestSubscriberAuthenticationDataCreate_CommonDBFails_RollsBack(t *testing.T) {
	rollbackCalled := false
	authDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
			return true, nil
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			rollbackCalled = true
			return nil
		},
	}
	commonDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
			return false, fmt.Errorf("common db failure")
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			return nil
		},
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	sub := DatabaseSubscriberAuthenticationData{}
	subsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "c9e8763286b5b9ffbdf56e1297d0887b",
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            "981d464c7c52eb6e5036234984ad0bcf",
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "5122250214c33e723a5dd523fc145fc0",
		},
		SequenceNumber: "16f3b3f70fc2",
	}
	err := sub.SubscriberAuthenticationDataCreate("imsi-1", &subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !rollbackCalled {
		t.Error("expected rollback to be called")
	}
}

func TestSubscriberAuthenticationDataDelete_Success(t *testing.T) {
	origAuth := map[string]interface{}{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]interface{}, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]interface{}) (bool, error) { return true, nil },
	}
	commonDB := &mockDB{
		deleteOneFunc: func(c string, f bson.M) error { return nil },
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	s := DatabaseSubscriberAuthenticationData{}
	err := s.SubscriberAuthenticationDataDelete("imsi-12345")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_AuthDBDeleteFails_Exits(t *testing.T) {
	origAuth := map[string]interface{}{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]interface{}, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return fmt.Errorf("fail on authdb delete") },
	}
	origAuthDB := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDB }()
	dbadapter.AuthDBClient = authDB

	s := DatabaseSubscriberAuthenticationData{}
	err := s.SubscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "fail on authdb delete") {
		t.Errorf("expected error about authdb delete, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails_RollbackSucceeds(t *testing.T) {
	origAuth := map[string]interface{}{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]interface{}, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]interface{}) (bool, error) { return true, nil },
	}
	commonDB := &mockDB{
		deleteOneFunc: func(c string, f bson.M) error { return fmt.Errorf("fail on commondb delete") },
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	s := DatabaseSubscriberAuthenticationData{}
	err := s.SubscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "amData delete failed, rolled back AuthDB change") {
		t.Errorf("expected error with rollback message, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails_RollbackFails(t *testing.T) {
	origAuth := map[string]interface{}{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]interface{}, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc: func(c string, f bson.M, data map[string]interface{}) (bool, error) {
			return false, fmt.Errorf("rollback fail")
		},
	}
	commonDB := &mockDB{
		deleteOneFunc: func(c string, f bson.M) error { return fmt.Errorf("fail on commondb delete") },
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	s := DatabaseSubscriberAuthenticationData{}
	err := s.SubscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "amData delete failed:") || !strings.Contains(err.Error(), "rollback failed") {
		t.Errorf("expected error with rollback fail message, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_NoDataInAuthDB_Exits(t *testing.T) {
	authDB := &mockDB{
		getOneFunc: func(c string, f bson.M) (map[string]interface{}, error) {
			return nil, fmt.Errorf("data not found in AuthDB")
		},
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]interface{}) (bool, error) { return true, nil },
	}
	commonDB := &mockDB{
		deleteOneFunc: func(c string, f bson.M) error { return fmt.Errorf("fail on commondb delete") },
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authDB
	dbadapter.CommonDBClient = commonDB

	s := DatabaseSubscriberAuthenticationData{}
	err := s.SubscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected data not found in AuthDB, got %v", err)
	}
}

func Test_handleSubscriberPost5G(t *testing.T) {
	origImsiData := ImsiData
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	origPostData := postData
	defer func() {
		ImsiData = origImsiData
		postData = origPostData
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
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
	ImsiData = make(map[string]*models.AuthenticationSubscription)
	dbadapter.AuthDBClient = &MockMongoPost{}
	dbadapter.CommonDBClient = &MockMongoPost{}
	postErr := handleSubscriberPost(ueId, configMsg.AuthSubData)
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
	if ImsiData[ueId] != nil {
		t.Errorf("Expected no ueId in memory, got %v", ImsiData[ueId])
	}
}

func Test_handleSubscriberDelete5G(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	origDeleteData := deleteData
	defer func() {
		deleteData = origDeleteData
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"

	deleteData = make([]map[string]interface{}, 0)
	dbadapter.AuthDBClient = &MockMongoDeleteOne{}
	dbadapter.CommonDBClient = &MockMongoDeleteOne{}
	delErr := handleSubscriberDelete(ueId)
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

func Test_handleSubscriberGet5G(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDBClient }()
	subscriberAuthData := DatabaseSubscriberAuthenticationData{}
	subscriber := models.AuthenticationSubscription{
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
	subscribers := []bson.M{configmodels.ToBsonM(subscriber)}
	subscribers[0]["ueId"] = "imsi-208930100007487"
	dbadapter.AuthDBClient = &MockMongoSubscriberGetOne{dbadapter.AuthDBClient, subscribers[0]}
	subscriberResult := subscriberAuthData.SubscriberAuthenticationDataGet("imsi-208930100007487")
	if !reflect.DeepEqual(&subscriber, subscriberResult) {
		t.Errorf("Expected subscriber %v, got %v", &subscriber, subscriberResult)
	}
}
