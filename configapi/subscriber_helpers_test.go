// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func authenticationSubscription() *models.AuthenticationSubscription {
	sqn := models.NewSequenceNumber()
	sqn.SetSqn("16f3b3f70fc2")
	authSubs := models.NewAuthenticationSubscription(models.AUTHMETHOD__5_G_AKA)
	authSubs.SetAuthenticationManagementField("8000")
	authSubs.SetEncOpcKey("8e27b6af0e692e750f32667a3b14605d")       // Required
	authSubs.SetEncPermanentKey("8baf473f2f8fd09487cccbd7097c6862") // Required
	authSubs.SetSequenceNumber(*sqn)
	return authSubs
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
	receivedPost   []map[string]any
	receivedPut    []map[string]any
	receivedDelete []map[string]any
	postErr        error
	putErr         error
	deleteErr      error
}

func (m *txMockDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.postErr != nil {
		return false, m.postErr
	}
	m.receivedPost = append(m.receivedPost, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.putErr != nil {
		return false, m.putErr
	}
	m.receivedPut = append(m.receivedPut, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   putData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.receivedDelete = append(m.receivedDelete, map[string]any{
		collKey:   collName,
		filterKey: filter,
	})
	return nil
}

// authSubsDataColl writes/deletes must go through AuthDBClient (its own server) while amData
// goes through CommonDBClient, as two independent non-transactional calls (see
// subscriberAuthenticationDataCreate/Update/Delete).
func TestSubscriberAuthenticationDataCreate_Success(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(authMock.receivedPost) != 1 {
		t.Fatalf("expected 1 auth DB post call, got %d", len(authMock.receivedPost))
	}
	if authMock.receivedPost[0][collKey] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, authMock.receivedPost[0][collKey])
	}
	if len(commonMock.receivedPost) != 1 {
		t.Fatalf("expected 1 amData post call, got %d", len(commonMock.receivedPost))
	}
	if commonMock.receivedPost[0][collKey] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, commonMock.receivedPost[0][collKey])
	}

	expectedFilter := bson.M{ueIdKey: testSubscriberImsi}
	if !reflect.DeepEqual(authMock.receivedPost[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authMock.receivedPost[0][filterKey])
	}
	if !reflect.DeepEqual(commonMock.receivedPost[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonMock.receivedPost[0][filterKey])
	}
}

func TestSubscriberAuthenticationDataCreate_AuthDBFails(t *testing.T) {
	authMock := &txMockDB{postErr: fmt.Errorf("auth db failure")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(commonMock.receivedPost) != 0 {
		t.Error("expected no CommonDB write when auth write fails")
	}
}

func TestSubscriberAuthenticationDataCreate_CommonDBFails(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{postErr: fmt.Errorf("common db failure")}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(authMock.receivedPost) != 1 {
		t.Error("expected auth write to have been attempted")
	}
}

func TestSubscriberAuthenticationDataDelete_Success(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(authMock.receivedDelete) != 1 {
		t.Fatalf("expected 1 auth DB delete call, got %d", len(authMock.receivedDelete))
	}
	if authMock.receivedDelete[0][collKey] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, authMock.receivedDelete[0][collKey])
	}
	if len(commonMock.receivedDelete) != 1 {
		t.Fatalf("expected 1 amData delete call, got %d", len(commonMock.receivedDelete))
	}
	if commonMock.receivedDelete[0][collKey] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, commonMock.receivedDelete[0][collKey])
	}

	expectedFilter := bson.M{ueIdKey: testSubscriberImsi}
	if !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authMock.receivedDelete[0][filterKey])
	}
	if !reflect.DeepEqual(commonMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonMock.receivedDelete[0][filterKey])
	}
}

func TestSubscriberAuthenticationDataDelete_AuthDBDeleteFails(t *testing.T) {
	authMock := &txMockDB{deleteErr: fmt.Errorf("fail on authdb delete")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(commonMock.receivedDelete) != 0 {
		t.Error("expected no CommonDB delete when auth delete fails")
	}
}

func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{deleteErr: fmt.Errorf("fail on commondb delete")}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(authMock.receivedDelete) != 1 {
		t.Error("expected auth delete to have been attempted")
	}
}

func Test_handleSubscriberGet(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDBClient }()
	subscriber := authenticationSubscription()
	dbadapter.AuthDBClient = &AuthDBMockDBClient{subscribers: []string{testSubscriberImsi}}
	subscriberResult := subscriberAuthenticationDataGet(testSubscriberImsi)
	if !reflect.DeepEqual(subscriber, subscriberResult) {
		t.Errorf("expected subscriber %v, got %v", subscriber, subscriberResult)
	}
}

// This write path fetches a device group straight from storage rather than through
// deviceGroupPostHelper, so it must relabel a rate unit left over from before that helper stored
// it as bps -- otherwise it persists the stale unit rather than correcting it.
func TestUpdateSubscriberInDeviceGroups_RelabelsStaleRateUnit(t *testing.T) {
	origDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origDBClient }()
	group := deviceGroup(testGroupName)
	mock := &DeviceGroupMockDBClient{configuredDeviceGroups: []configmodels.DeviceGroups{group}}
	dbadapter.CommonDBClient = mock

	statusCode, err := updateSubscriberInDeviceGroups(group.Imsis[0])
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, statusCode)
	}
	if len(mock.postData) == 0 {
		t.Fatal("expected the device group to be stored")
	}
	var stored configmodels.DeviceGroups
	if err := json.Unmarshal(configmodels.MapToByte(mock.postData[0][dataKey].(map[string]any)), &stored); err != nil {
		t.Fatalf("failed to unmarshal the stored device group: %v", err)
	}
	qos := stored.IpDomainsExpanded[0].UeDnnQos
	if qos.BitrateUnit != bitrateUnitBps {
		t.Errorf("stored bitrate-unit = %q, want %q", qos.BitrateUnit, bitrateUnitBps)
	}
	// The rates themselves are untouched: this path does not re-run convertToBps.
	original := group.IpDomainsExpanded[0].UeDnnQos
	if qos.DnnMbrUplink != original.DnnMbrUplink || qos.DnnMbrDownlink != original.DnnMbrDownlink {
		t.Errorf("rates changed: got %d/%d, want %d/%d", qos.DnnMbrUplink, qos.DnnMbrDownlink, original.DnnMbrUplink, original.DnnMbrDownlink)
	}
}
