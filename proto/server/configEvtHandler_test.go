// SPDX-FileCopyrightText: 2023 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	execCommandTimesCalled = 0
	postData               []map[string]interface{}
	deleteData             []map[string]interface{}
)

func deviceGroup(name string) configmodels.DeviceGroups {
	traffic_class := configmodels.TrafficClassInfo{
		Name: "platinum",
		Qci:  8,
		Arp:  6,
		Pdb:  300,
		Pelr: 6,
	}
	qos := configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		DnnMbrUplink:   10000000,
		DnnMbrDownlink: 10000000,
		BitrateUnit:    "kbps",
		TrafficClass:   &traffic_class,
	}
	ipdomain := configmodels.DeviceGroupsIpDomainExpanded{
		Dnn:          "internet",
		UeIpPool:     "172.250.1.0/16",
		DnsPrimary:   "1.1.1.1",
		DnsSecondary: "8.8.8.8",
		Mtu:          1460,
		UeDnnQos:     &qos,
	}
	deviceGroup := configmodels.DeviceGroups{
		DeviceGroupName:  name,
		Imsis:            []string{"1234", "5678"},
		SiteInfo:         "demo",
		IpDomainName:     "pool1",
		IpDomainExpanded: ipdomain,
	}
	return deviceGroup
}

type MockMongoPost struct {
	dbadapter.DBInterface
}

type MockMongoDeleteOne struct {
	dbadapter.DBInterface
}

type MockMongoGetOneNil struct {
	dbadapter.DBInterface
}

type MockMongoGetManyNil struct {
	dbadapter.DBInterface
}

type MockMongoGetManyGroups struct {
	dbadapter.DBInterface
}

type MockMongoGetManySlices struct {
	dbadapter.DBInterface
}

type MockMongoDeviceGroupGetOne struct {
	dbadapter.DBInterface
	testGroup configmodels.DeviceGroups
}

type MockMongoSliceGetOne struct {
	dbadapter.DBInterface
	testSlice configmodels.Slice
}

type MockMongoSubscriberGetOne struct {
	dbadapter.DBInterface
	testSubscriber bson.M
}

func (m *MockMongoPost) RestfulAPIPost(coll string, filter primitive.M, data map[string]interface{}) (bool, error) {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true, nil
}

func (m *MockMongoDeleteOne) RestfulAPIDeleteOne(coll string, filter primitive.M) error {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
	}
	deleteData = append(deleteData, params)
	return nil
}

func (m *MockMongoDeleteOne) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockMongoGetOneNil) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var value map[string]interface{}
	return value, nil
}

func (m *MockMongoDeviceGroupGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousGroupBson bson.M
	previousGroup, err := json.Marshal(m.testGroup)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousGroup, &previousGroupBson)
	if err != nil {
		return nil, err
	}
	return previousGroupBson, nil
}

func (m *MockMongoSliceGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousSliceBson bson.M
	previousSlice, err := json.Marshal(m.testSlice)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousSlice, &previousSliceBson)
	if err != nil {
		return nil, err
	}
	return previousSliceBson, nil
}

func (m *MockMongoSubscriberGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousSubscriberBson bson.M
	previousSubscriber, err := json.Marshal(m.testSubscriber)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousSubscriber, &previousSubscriberBson)
	if err != nil {
		return nil, err
	}
	return previousSubscriberBson, nil
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", "ENTER YOUR COMMAND HERE"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	execCommandTimesCalled += 1
	return cmd
}

func networkSlice(name string) configmodels.Slice {
	upf := make(map[string]interface{}, 0)
	upf["upf-name"] = "upf"
	upf["upf-port"] = "8805"
	plmn := configmodels.SliceSiteInfoPlmn{
		Mcc: "208",
		Mnc: "93",
	}
	gnodeb := configmodels.SliceSiteInfoGNodeBs{
		Name: "demo-gnb1",
		Tac:  1,
	}
	slice_id := configmodels.SliceSliceId{
		Sst: "1",
		Sd:  "010203",
	}
	site_info := configmodels.SliceSiteInfo{
		SiteName: "demo",
		Plmn:     plmn,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{gnodeb},
		Upf:      upf,
	}
	slice := configmodels.Slice{
		SliceName:       name,
		SliceId:         slice_id,
		SiteDeviceGroup: []string{"group1", "group2"},
		SiteInfo:        site_info,
	}
	return slice
}

func Test_handleSubscriberGet5G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { subscriberAuthData = origSubscriberAuthData; dbadapter.AuthDBClient = origAuthDBClient }()
	subscriberAuthData = configapi.DatabaseSubscriberAuthenticationData{}
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

type InMemoryAuthDataStore struct {
	store map[string]*models.AuthenticationSubscription
}

func (m *InMemoryAuthDataStore) SubscriberAuthenticationDataGet(imsi string) *models.AuthenticationSubscription {
	return m.store[imsi]
}

func (m *InMemoryAuthDataStore) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	m.store[imsi] = authSubData
	return nil
}

func (m *InMemoryAuthDataStore) SubscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	m.store[imsi] = authSubData
	return nil
}

func (m *InMemoryAuthDataStore) SubscriberAuthenticationDataDelete(imsi string) error {
	delete(m.store, imsi)
	return nil
}

func Test_handleSubscriberGet4G(t *testing.T) {
	expected := &models.AuthenticationSubscription{
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

	imsi := "imsi-208930100007487"
	
	orig := subscriberAuthData
	defer func() { subscriberAuthData = orig }()

	subscriberAuthData = &InMemoryAuthDataStore{
		store: map[string]*models.AuthenticationSubscription{
			imsi: expected,
		},
	}

	actual := subscriberAuthData.SubscriberAuthenticationDataGet(imsi)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected subscriber:\n%+v\nGot:\n%+v", expected, actual)
	}
}

func (m *MockMongoGetManyNil) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	var value []map[string]interface{}
	return value, nil
}

func (m *MockMongoGetManyGroups) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	testGroup := deviceGroup("testGroup")
	var previousGroupBson bson.M
	previousGroup, err := json.Marshal(testGroup)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousGroup, &previousGroupBson)
	if err != nil {
		return nil, err
	}
	var groups []map[string]interface{}
	groups = append(groups, previousGroupBson)
	return groups, nil
}

func (m *MockMongoGetManySlices) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	testSlice := networkSlice("testGroup")
	var previousSliceBson bson.M
	previousSlice, err := json.Marshal(testSlice)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousSlice, &previousSliceBson)
	if err != nil {
		return nil, err
	}
	var slices []map[string]interface{}
	slices = append(slices, previousSliceBson)
	return slices, nil
}

func Test_firstConfigReceived_noConfigInDB(t *testing.T) {
	dbadapter.CommonDBClient = &MockMongoGetManyNil{}
	result := firstConfigReceived()
	if result {
		t.Errorf("Expected firstConfigReceived to return false, got %v", result)
	}
}

func Test_firstConfigReceived_deviceGroupInDB(t *testing.T) {
	dbadapter.CommonDBClient = &MockMongoGetManyGroups{}
	result := firstConfigReceived()
	if !result {
		t.Errorf("Expected firstConfigReceived to return true, got %v", result)
	}
}

func Test_firstConfigReceived_sliceInDB(t *testing.T) {
	dbadapter.CommonDBClient = &MockMongoGetManySlices{}
	result := firstConfigReceived()
	if !result {
		t.Errorf("Expected firstConfigReceived to return true, got %v", result)
	}
}
