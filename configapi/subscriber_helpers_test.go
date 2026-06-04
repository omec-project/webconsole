// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func authenticationSubscription() *models.AuthenticationSubscription {
	return &models.AuthenticationSubscription{
		AuthenticationManagementField: openapi.PtrString("8000"),
		AuthenticationMethod:          "5G_AKA",
		EncOpcKey:                     openapi.PtrString("8e27b6af0e692e750f32667a3b14605d"), // Required
		EncPermanentKey:               openapi.PtrString("8baf473f2f8fd09487cccbd7097c6862"), // Required
		SequenceNumber: &models.SequenceNumber{
			Sqn: openapi.PtrString("16f3b3f70fc2"),
		},
	}
}

const testAuthDbName = "auth_test_db"

func setupTestFactory() func() {
	origConfig := factory.WebUIConfig
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			Mongodb: &factory.Mongodb{
				AuthKeysDbName: testAuthDbName,
			},
		},
	}
	return func() {
		factory.WebUIConfig = origConfig
	}
}

type txMockDB struct {
	dbadapter.DBInterface
	receivedPostOnDB      []map[string]any
	receivedPostWithCtx   []map[string]any
	receivedPutOneOnDB    []map[string]any
	receivedPutOneWithCtx []map[string]any
	receivedDeleteOnDB    []map[string]any
	receivedDeleteWithCtx []map[string]any
	postOnDBErr           error
	postWithCtxErr        error
	putOneOnDBErr         error
	putOneWithCtxErr      error
	deleteOnDBErr         error
	deleteWithCtxErr      error
}

func (m *txMockDB) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

func (m *txMockDB) RestfulAPIPostOnDB(ctx context.Context, dbName string, collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.postOnDBErr != nil {
		return false, m.postOnDBErr
	}
	m.receivedPostOnDB = append(m.receivedPostOnDB, map[string]any{
		"dbName": dbName,
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIPostWithContext(ctx context.Context, collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.postWithCtxErr != nil {
		return false, m.postWithCtxErr
	}
	m.receivedPostWithCtx = append(m.receivedPostWithCtx, map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIPutOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.putOneOnDBErr != nil {
		return false, m.putOneOnDBErr
	}
	m.receivedPutOneOnDB = append(m.receivedPutOneOnDB, map[string]any{
		"dbName": dbName,
		"coll":   collName,
		"filter": filter,
		"data":   putData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIPutOneWithContext(ctx context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.putOneWithCtxErr != nil {
		return false, m.putOneWithCtxErr
	}
	m.receivedPutOneWithCtx = append(m.receivedPutOneWithCtx, map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   putData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIDeleteOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M) error {
	if m.deleteOnDBErr != nil {
		return m.deleteOnDBErr
	}
	m.receivedDeleteOnDB = append(m.receivedDeleteOnDB, map[string]any{
		"dbName": dbName,
		"coll":   collName,
		"filter": filter,
	})
	return nil
}

func (m *txMockDB) RestfulAPIDeleteOneWithContext(ctx context.Context, collName string, filter bson.M) error {
	if m.deleteWithCtxErr != nil {
		return m.deleteWithCtxErr
	}
	m.receivedDeleteWithCtx = append(m.receivedDeleteWithCtx, map[string]any{
		"coll":   collName,
		"filter": filter,
	})
	return nil
}

func TestSubscriberAuthenticationDataCreate_Success(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate("imsi-1", subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.receivedPostOnDB) != 1 {
		t.Fatalf("expected 1 PostOnDB call, got %d", len(mock.receivedPostOnDB))
	}
	if mock.receivedPostOnDB[0]["dbName"] != testAuthDbName {
		t.Errorf("expected auth DB name %q, got %q", testAuthDbName, mock.receivedPostOnDB[0]["dbName"])
	}
	if mock.receivedPostOnDB[0]["coll"] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, mock.receivedPostOnDB[0]["coll"])
	}
	if len(mock.receivedPostWithCtx) != 1 {
		t.Fatalf("expected 1 PostWithContext call, got %d", len(mock.receivedPostWithCtx))
	}
	if mock.receivedPostWithCtx[0]["coll"] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, mock.receivedPostWithCtx[0]["coll"])
	}
}

func TestSubscriberAuthenticationDataCreate_AuthDBFails_TransactionAborts(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{
		postOnDBErr: fmt.Errorf("auth db failure"),
	}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate("imsi-1", subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(mock.receivedPostWithCtx) != 0 {
		t.Error("expected no CommonDB write when auth write fails")
	}
}

func TestSubscriberAuthenticationDataCreate_CommonDBFails_TransactionAborts(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{
		postWithCtxErr: fmt.Errorf("common db failure"),
	}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate("imsi-1", subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(mock.receivedPostOnDB) != 1 {
		t.Error("expected auth write to have been attempted")
	}
}

func TestSubscriberAuthenticationDataDelete_Success(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(mock.receivedDeleteOnDB) != 1 {
		t.Fatalf("expected 1 DeleteOneOnDB call, got %d", len(mock.receivedDeleteOnDB))
	}
	if mock.receivedDeleteOnDB[0]["dbName"] != testAuthDbName {
		t.Errorf("expected auth DB name %q, got %q", testAuthDbName, mock.receivedDeleteOnDB[0]["dbName"])
	}
	if mock.receivedDeleteOnDB[0]["coll"] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, mock.receivedDeleteOnDB[0]["coll"])
	}
	if len(mock.receivedDeleteWithCtx) != 1 {
		t.Fatalf("expected 1 DeleteOneWithContext call, got %d", len(mock.receivedDeleteWithCtx))
	}
	if mock.receivedDeleteWithCtx[0]["coll"] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, mock.receivedDeleteWithCtx[0]["coll"])
	}
}

func TestSubscriberAuthenticationDataDelete_AuthDBDeleteFails_TransactionAborts(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{
		deleteOnDBErr: fmt.Errorf("fail on authdb delete"),
	}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(mock.receivedDeleteWithCtx) != 0 {
		t.Error("expected no CommonDB delete when auth delete fails")
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails_TransactionAborts(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	mock := &txMockDB{
		deleteWithCtxErr: fmt.Errorf("fail on commondb delete"),
	}
	origCommonDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDB }()
	dbadapter.CommonDBClient = mock

	err := subscriberAuthenticationDataDelete("imsi-12345")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(mock.receivedDeleteOnDB) != 1 {
		t.Error("expected auth delete to have been attempted")
	}
}

func Test_handleSubscriberPost(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()
	origCommonDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDBClient }()
	ueId := "imsi-208930100007487"
	authSubData := authenticationSubscription()
	mock := &txMockDB{}
	dbadapter.CommonDBClient = mock
	postErr := subscriberAuthenticationDataCreate(ueId, authSubData)
	if postErr != nil {
		t.Errorf("could not handle subscriber post: %v", postErr)
	}
	if len(mock.receivedPostOnDB) != 1 {
		t.Fatalf("expected 1 PostOnDB call, got %d", len(mock.receivedPostOnDB))
	}
	if mock.receivedPostOnDB[0]["coll"] != authSubsDataColl {
		t.Errorf("expected collection %v, got %v", authSubsDataColl, mock.receivedPostOnDB[0]["coll"])
	}
	if mock.receivedPostOnDB[0]["dbName"] != testAuthDbName {
		t.Errorf("expected dbName %v, got %v", testAuthDbName, mock.receivedPostOnDB[0]["dbName"])
	}
	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(mock.receivedPostOnDB[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, mock.receivedPostOnDB[0]["filter"])
	}

	var authSubResult models.AuthenticationSubscription
	result := mock.receivedPostOnDB[0]["data"].(map[string]any)
	err := json.Unmarshal(configmodels.MapToByte(result), &authSubResult)
	if err != nil {
		t.Errorf("could not unmarshall result %v", result)
	}
	if len(mock.receivedPostWithCtx) != 1 {
		t.Fatalf("expected 1 PostWithContext call, got %d", len(mock.receivedPostWithCtx))
	}
	if mock.receivedPostWithCtx[0]["coll"] != amDataColl {
		t.Errorf("expected collection %v, got %v", amDataColl, mock.receivedPostWithCtx[0]["coll"])
	}
	if !reflect.DeepEqual(mock.receivedPostWithCtx[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, mock.receivedPostWithCtx[0]["filter"])
	}
	amDataResult := mock.receivedPostWithCtx[0]["data"].(map[string]any)
	if amDataResult["ueId"] != ueId {
		t.Errorf("expected ueId %v, got %v", ueId, amDataResult["ueId"])
	}
}

func Test_handleSubscriberDelete(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()
	origCommonDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommonDBClient }()
	ueId := "imsi-208930100007487"
	mock := &txMockDB{}
	dbadapter.CommonDBClient = mock
	delErr := subscriberAuthenticationDataDelete(ueId)
	if delErr != nil {
		t.Errorf("could not handle subscriber delete: %v", delErr)
	}
	if len(mock.receivedDeleteOnDB) != 1 {
		t.Fatalf("expected 1 DeleteOneOnDB call, got %d", len(mock.receivedDeleteOnDB))
	}
	if mock.receivedDeleteOnDB[0]["coll"] != authSubsDataColl {
		t.Errorf("expected collection %v, got %v", authSubsDataColl, mock.receivedDeleteOnDB[0]["coll"])
	}
	if mock.receivedDeleteOnDB[0]["dbName"] != testAuthDbName {
		t.Errorf("expected dbName %v, got %v", testAuthDbName, mock.receivedDeleteOnDB[0]["dbName"])
	}

	expectedFilter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(mock.receivedDeleteOnDB[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, mock.receivedDeleteOnDB[0]["filter"])
	}
	if len(mock.receivedDeleteWithCtx) != 1 {
		t.Fatalf("expected 1 DeleteOneWithContext call, got %d", len(mock.receivedDeleteWithCtx))
	}
	if mock.receivedDeleteWithCtx[0]["coll"] != amDataColl {
		t.Errorf("expected collection %v, got %v", amDataColl, mock.receivedDeleteWithCtx[0]["coll"])
	}
	if !reflect.DeepEqual(mock.receivedDeleteWithCtx[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, mock.receivedDeleteWithCtx[0]["filter"])
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
