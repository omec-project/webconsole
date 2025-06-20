// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/omec-project/openapi/models"
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
	err := sub.SubscriberAuthenticationDataCreate("imsi-1", &models.AuthenticationSubscription{})
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
	err := sub.SubscriberAuthenticationDataCreate("imsi-1", &models.AuthenticationSubscription{})
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
