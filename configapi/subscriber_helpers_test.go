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

func authenticationSubscription() *models.AuthenticationSubscription {
	return &models.AuthenticationSubscription{
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
			EncryptionKey:       "",
			PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
		},
		SequenceNumber: "16f3b3f70fc2",
	}
}

type mockDB struct {
	getOneFunc    func(collName string, filter bson.M) (map[string]any, error)
	deleteOneFunc func(collName string, filter bson.M) error
	postFunc      func(collName string, filter bson.M, postData map[string]any) (bool, error)
	dbadapter.DBInterface
}

func (m *mockDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	return m.getOneFunc(collName, filter)
}

func (m *mockDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return m.postFunc(collName, filter, postData)
}

func (m *mockDB) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return m.deleteOneFunc(collName, filter)
}

func TestSubscriberAuthenticationDataCreate_Success(t *testing.T) {
	authCalled, commonCalled := false, false

	authDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]any) (bool, error) {
			authCalled = true
			return true, nil
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			t.Error("rollback should not be called on success")
			return nil
		},
	}

	commonDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]any) (bool, error) {
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

	subsData := authenticationSubscription()
	err := SubscriberAuthenticationDataCreate("imsi-1", subsData)
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
		postFunc: func(coll string, filter bson.M, data map[string]any) (bool, error) {
			return true, nil
		},
		deleteOneFunc: func(coll string, filter bson.M) error {
			rollbackCalled = true
			return nil
		},
	}
	commonDB := &mockDB{
		postFunc: func(coll string, filter bson.M, data map[string]any) (bool, error) {
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

	subsData := authenticationSubscription()
	err := SubscriberAuthenticationDataCreate("imsi-1", subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !rollbackCalled {
		t.Error("expected rollback to be called")
	}
}

func TestSubscriberAuthenticationDataDelete_Success(t *testing.T) {
	origAuth := map[string]any{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]any, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]any) (bool, error) { return true, nil },
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

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_AuthDBDeleteFails_Exits(t *testing.T) {
	origAuth := map[string]any{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]any, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return fmt.Errorf("fail on authdb delete") },
	}
	origAuthDB := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDB }()
	dbadapter.AuthDBClient = authDB

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "fail on authdb delete") {
		t.Errorf("expected error about authdb delete, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails_RollbackSucceeds(t *testing.T) {
	origAuth := map[string]any{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]any, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]any) (bool, error) { return true, nil },
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

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "amData delete failed, rolled back AuthDB change") {
		t.Errorf("expected error with rollback message, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails_RollbackFails(t *testing.T) {
	origAuth := map[string]any{"ueId": "imsi-12345"}
	authDB := &mockDB{
		getOneFunc:    func(c string, f bson.M) (map[string]any, error) { return origAuth, nil },
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc: func(c string, f bson.M, data map[string]any) (bool, error) {
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

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "amData delete failed:") || !strings.Contains(err.Error(), "rollback failed") {
		t.Errorf("expected error with rollback fail message, got %v", err)
	}
}

func TestSubscriberAuthenticationDataDelete_NoDataInAuthDB_Exits(t *testing.T) {
	authDB := &mockDB{
		getOneFunc: func(c string, f bson.M) (map[string]any, error) {
			return nil, fmt.Errorf("data not found in AuthDB")
		},
		deleteOneFunc: func(c string, f bson.M) error { return nil },
		postFunc:      func(c string, f bson.M, data map[string]any) (bool, error) { return true, nil },
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

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected data not found in AuthDB, got %v", err)
	}
}

func Test_handleSubscriberPost(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"
	authSubData := authenticationSubscription()
	authDbClientMock := &AuthDBMockDBClient{}
	dbadapter.AuthDBClient = authDbClientMock
	commonDbClientMock := &PostSubscriberMockDBClient{}
	dbadapter.CommonDBClient = commonDbClientMock
	postErr := SubscriberAuthenticationDataCreate(ueId, authSubData)
	if postErr != nil {
		t.Errorf("could not handle subscriber post: %v", postErr)
	}
	expectedAuthSubCollection := AuthSubsDataColl
	expectedAmDataCollection := AmDataColl
	if authDbClientMock.receivedPostData[0]["coll"] != expectedAuthSubCollection {
		t.Errorf("expected collection %v, got %v", expectedAuthSubCollection, authDbClientMock.receivedPostData[0]["coll"])
	}
	if commonDbClientMock.receivedPostData[0]["coll"] != expectedAmDataCollection {
		t.Errorf("expected collection %v, got %v", expectedAmDataCollection, commonDbClientMock.receivedPostData[0]["coll"])
	}

	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(authDbClientMock.receivedPostData[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authDbClientMock.receivedPostData[0]["filter"])
	}
	if !reflect.DeepEqual(commonDbClientMock.receivedPostData[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonDbClientMock.receivedPostData[0]["filter"])
	}

	var authSubResult models.AuthenticationSubscription
	result := authDbClientMock.receivedPostData[0]["data"].(map[string]any)
	err := json.Unmarshal(configmodels.MapToByte(result), &authSubResult)
	if err != nil {
		t.Errorf("could not unmarshall result %v", result)
	}
	amDataResult := commonDbClientMock.receivedPostData[0]["data"].(map[string]any)
	if amDataResult["ueId"] != ueId {
		t.Errorf("expected ueId %v, got %v", ueId, amDataResult["ueId"])
	}
}

func Test_handleSubscriberDelete(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	origCommonDBClient := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDBClient
		dbadapter.CommonDBClient = origCommonDBClient
	}()
	ueId := "imsi-208930100007487"

	authDbClientMock := &AuthDBMockDBClient{}
	dbadapter.AuthDBClient = authDbClientMock
	commonDbClientMock := &DeleteSubscriberMockDBClient{}
	dbadapter.CommonDBClient = commonDbClientMock
	delErr := subscriberAuthenticationDataDelete(ueId)
	if delErr != nil {
		t.Errorf("could not handle subscriber delete: %v", delErr)
	}
	expectedAuthSubCollection := AuthSubsDataColl
	expectedAmDataCollection := AmDataColl
	if authDbClientMock.deleteData[0]["coll"] != expectedAuthSubCollection {
		t.Errorf("expected collection %v, got %v", expectedAuthSubCollection, authDbClientMock.deleteData[0]["coll"])
	}
	if commonDbClientMock.deleteData[0]["coll"] != expectedAmDataCollection {
		t.Errorf("expected collection %v, got %v", expectedAmDataCollection, commonDbClientMock.deleteData[0]["coll"])
	}

	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(authDbClientMock.deleteData[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authDbClientMock.deleteData[0]["filter"])
	}
	if !reflect.DeepEqual(commonDbClientMock.deleteData[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonDbClientMock.deleteData[0]["filter"])
	}
}

func Test_handleSubscriberGet(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDBClient }()
	subscriber := authenticationSubscription()
	dbadapter.AuthDBClient = &AuthDBMockDBClient{subscribers: []string{"imsi-208930100007487"}}
	subscriberResult := subscriberAuthenticationDataGet("imsi-208930100007487")
	if !reflect.DeepEqual(subscriber, subscriberResult) {
		t.Errorf("expected subscriber %v, got %v", &subscriber, subscriberResult)
	}
}
