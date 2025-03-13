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
	"github.com/omec-project/webconsole/backend/factory"
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

func Test_sendPebbleNotification_on_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	numPebbleNotificationsSent := execCommandTimesCalled
	networkSlice := []configmodels.Slice{networkSlice("slice1")}
	factory.WebUIConfig.Configuration.SendPebbleNotifications = true

	configMsg := configmodels.ConfigMessage{
		SliceName: networkSlice[0].SliceName,
		Slice:     &networkSlice[0],
	}
	subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
	dbadapter.CommonDBClient = &MockMongoPost{dbadapter.CommonDBClient}
	dbadapter.CommonDBClient = &MockMongoGetOneNil{dbadapter.CommonDBClient}
	handleNetworkSlicePost(&configMsg, subsUpdateChan)

	if execCommandTimesCalled != numPebbleNotificationsSent+1 {
		t.Errorf("Unexpected number of Pebble notifications: %v. Should be: %v", execCommandTimesCalled, numPebbleNotificationsSent+1)
	}
}

func Test_sendPebbleNotification_off_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	numPebbleNotificationsSent := execCommandTimesCalled
	networkSlices := []configmodels.Slice{networkSlice("slice1")}
	factory.WebUIConfig.Configuration.SendPebbleNotifications = false

	for _, testSlice := range networkSlices {
		configMsg := configmodels.ConfigMessage{
			SliceName: testSlice.SliceName,
			Slice:     &testSlice,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		dbadapter.CommonDBClient = &MockMongoPost{dbadapter.CommonDBClient}
		dbadapter.CommonDBClient = &MockMongoGetOneNil{dbadapter.CommonDBClient}
		handleNetworkSlicePost(&configMsg, subsUpdateChan)
	}

	if execCommandTimesCalled != numPebbleNotificationsSent {
		t.Errorf("Unexpected number of Pebble notifications: %v. Should be: %v", execCommandTimesCalled, numPebbleNotificationsSent)
	}
}

func Test_handleDeviceGroupPost(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{deviceGroup("group1"), deviceGroup("group2"), deviceGroup("group_no_imsis"), deviceGroup("group_no_traf_class"), deviceGroup("group_no_qos")}
	deviceGroups[2].Imsis = []string{}
	deviceGroups[3].IpDomainExpanded.UeDnnQos.TrafficClass = nil
	deviceGroups[4].IpDomainExpanded.UeDnnQos = nil
	factory.WebUIConfig.Configuration.Mode5G = true
	for _, testGroup := range deviceGroups {
		configMsg := configmodels.ConfigMessage{
			DevGroupName: testGroup.DeviceGroupName,
			DevGroup:     &testGroup,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		postData = make([]map[string]interface{}, 0)
		dbadapter.CommonDBClient = &(MockMongoPost{dbadapter.CommonDBClient})
		dbadapter.CommonDBClient = &MockMongoGetOneNil{dbadapter.CommonDBClient}
		handleDeviceGroupPost(&configMsg, subsUpdateChan)
		expected_collection := devGroupDataColl
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"group-name": testGroup.DeviceGroupName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultGroup configmodels.DeviceGroups
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		err := json.Unmarshal(configmodels.MapToByte(result), &resultGroup)
		if err != nil {
			t.Errorf("Could not unmarshall result %v", result)
		}
		if !reflect.DeepEqual(resultGroup, testGroup) {
			t.Errorf("Expected group %v, got %v", testGroup, resultGroup)
		}
		receivedConfigMsg := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if receivedConfigMsg.PrevDevGroup.DeviceGroupName != "" {
			t.Errorf("Expected previous device group name to be empty, got %v", receivedConfigMsg.PrevDevGroup.DeviceGroupName)
		}
	}
}

func Test_handleDeviceGroupPost_alreadyExists(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{deviceGroup("group1"), deviceGroup("group2"), deviceGroup("group_no_imsis"), deviceGroup("group_no_traf_class"), deviceGroup("group_no_qos")}
	deviceGroups[2].Imsis = []string{}
	deviceGroups[3].IpDomainExpanded.UeDnnQos.TrafficClass = nil
	deviceGroups[4].IpDomainExpanded.UeDnnQos = nil
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testGroup := range deviceGroups {
		configMsg := configmodels.ConfigMessage{
			DevGroupName: testGroup.DeviceGroupName,
			DevGroup:     &testGroup,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		postData = make([]map[string]interface{}, 0)
		dbadapter.CommonDBClient = &MockMongoPost{dbadapter.CommonDBClient}
		dbadapter.CommonDBClient = &(MockMongoDeviceGroupGetOne{dbadapter.CommonDBClient, testGroup})
		handleDeviceGroupPost(&configMsg, subsUpdateChan)
		expected_collection := devGroupDataColl
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"group-name": testGroup.DeviceGroupName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultGroup configmodels.DeviceGroups
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		err := json.Unmarshal(configmodels.MapToByte(result), &resultGroup)
		if err != nil {
			t.Errorf("Could not unmarshall result %v", result)
		}
		if !reflect.DeepEqual(resultGroup, testGroup) {
			t.Errorf("Expected group %v, got %v", testGroup, resultGroup)
		}
		receivedConfigMsg := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if !reflect.DeepEqual(receivedConfigMsg.PrevDevGroup, &testGroup) {
			t.Errorf("Expected previous device group to be %v, got %v", testGroup, receivedConfigMsg.PrevDevGroup)
		}
	}
}

func Test_handleDeviceGroupDelete(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{deviceGroup("group1")}
	factory.WebUIConfig.Configuration.Mode5G = true
	for _, testGroup := range deviceGroups {
		configMsg := configmodels.ConfigMessage{
			MsgType:      configmodels.Device_group,
			MsgMethod:    configmodels.Delete_op,
			DevGroupName: testGroup.DeviceGroupName,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		deleteData = make([]map[string]interface{}, 0)
		dbadapter.CommonDBClient = &MockMongoDeleteOne{dbadapter.CommonDBClient}
		dbadapter.CommonDBClient = &(MockMongoDeviceGroupGetOne{dbadapter.CommonDBClient, testGroup})
		handleDeviceGroupDelete(&configMsg, subsUpdateChan)
		expected_collection := devGroupDataColl
		if deleteData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, deleteData[0]["coll"])
		}
		expected_filter := bson.M{"group-name": testGroup.DeviceGroupName}
		if !reflect.DeepEqual(deleteData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, deleteData[0]["filter"])
		}
		receivedConfigMsg := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if !reflect.DeepEqual(receivedConfigMsg.PrevDevGroup, &testGroup) {
			t.Errorf("Expected previous device group to be %v, got %v", testGroup, receivedConfigMsg.PrevDevGroup)
		}
	}
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

func Test_handleNetworkSlicePost(t *testing.T) {
	networkSlices := []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"), networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups")}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testSlice := range networkSlices {
		configMsg := configmodels.ConfigMessage{
			SliceName: testSlice.SliceName,
			Slice:     &testSlice,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		postData = make([]map[string]interface{}, 0)
		dbadapter.CommonDBClient = &MockMongoPost{dbadapter.CommonDBClient}
		dbadapter.CommonDBClient = &MockMongoGetOneNil{dbadapter.CommonDBClient}
		handleNetworkSlicePost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.sliceData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"slice-name": testSlice.SliceName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultSlice configmodels.Slice
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		err := json.Unmarshal(configmodels.MapToByte(result), &resultSlice)
		if err != nil {
			t.Errorf("Could not unmarshal result %v", result)
		}
		if !reflect.DeepEqual(resultSlice, testSlice) {
			t.Errorf("Expected slice %v, got %v", testSlice, resultSlice)
		}
		receivedConfigMsg := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if receivedConfigMsg.PrevSlice.SliceName != "" {
			t.Errorf("Expected previous network slice name to be empty, got %v", receivedConfigMsg.PrevSlice.SliceName)
		}
	}
}

func Test_handleNetworkSlicePost_alreadyExists(t *testing.T) {
	networkSlices := []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"), networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups")}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testSlice := range networkSlices {
		configMsg := configmodels.ConfigMessage{
			SliceName: testSlice.SliceName,
			Slice:     &testSlice,
		}
		subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
		postData = make([]map[string]interface{}, 0)
		var previousSliceBson bson.M
		previousSlice, err := json.Marshal(testSlice)
		if err != nil {
			t.Errorf("Could not marshal result %v", testSlice)
		}
		err = json.Unmarshal(previousSlice, &previousSliceBson)
		if err != nil {
			t.Errorf("Could not unmarshal previousSlice %v", previousSlice)
		}
		dbadapter.CommonDBClient = &MockMongoPost{dbadapter.CommonDBClient}
		dbadapter.CommonDBClient = &MockMongoSliceGetOne{dbadapter.CommonDBClient, testSlice}
		handleNetworkSlicePost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.sliceData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"slice-name": testSlice.SliceName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultSlice configmodels.Slice
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		err = json.Unmarshal(configmodels.MapToByte(result), &resultSlice)
		if err != nil {
			t.Errorf("Could not unmarshal result %v", result)
		}
		if !reflect.DeepEqual(resultSlice, testSlice) {
			t.Errorf("Expected slice %v, got %v", testSlice, resultSlice)
		}
		receivedConfigMsg := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if !reflect.DeepEqual(receivedConfigMsg.PrevSlice, &testSlice) {
			t.Errorf("Expected previous network slice to be %v, got %v", testSlice, receivedConfigMsg.PrevSlice)
		}
	}
}

func Test_handleSubscriberGet5G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { subscriberAuthData = origSubscriberAuthData; dbadapter.AuthDBClient = origAuthDBClient }()
	subscriberAuthData = DatabaseSubscriberAuthenticationData{}
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

func Test_handleSubscriberGet4G(t *testing.T) {
	origSubscriberAuthData := subscriberAuthData
	origImsiData := imsiData
	defer func() { subscriberAuthData = origSubscriberAuthData; imsiData = origImsiData }()
	subscriberAuthData = MemorySubscriberAuthenticationData{}
	imsiData = make(map[string]*models.AuthenticationSubscription)
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
	imsiData["imsi-208930100007487"] = &subscriber
	subscriberResult := subscriberAuthData.SubscriberAuthenticationDataGet("imsi-208930100007487")
	if !reflect.DeepEqual(&subscriber, subscriberResult) {
		t.Errorf("Expected subscriber %v, got %v", &subscriber, subscriberResult)
	}
}

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
	handleSubscriberPost(ueId, configMsg.AuthSubData)

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
	var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
	err := json.Unmarshal(configmodels.MapToByte(result), &authSubResult)
	if err != nil {
		t.Errorf("Could not unmarshall result %v", result)
	}
	if !reflect.DeepEqual(configMsg.AuthSubData, &authSubResult) {
		t.Errorf("Expected authSubData %v, got %v", configMsg.AuthSubData, &authSubResult)
	}
	var amDataResult map[string]interface{} = postData[1]["data"].(map[string]interface{})
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
	handleSubscriberPost(ueId, configMsg.AuthSubData)

	expectedAmDataCollection := amDataColl
	if postData[0]["coll"] != expectedAmDataCollection {
		t.Errorf("Expected collection %v, got %v", expectedAmDataCollection, postData[0]["coll"])
	}

	expected_filter := bson.M{"ueId": ueId}
	if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
		t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
	}

	var AmDataResult map[string]interface{} = postData[0]["data"].(map[string]interface{})
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
	handleSubscriberDelete(ueId)

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
	handleSubscriberDelete(ueId)

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
