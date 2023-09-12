// SPDX-FileCopyrightText: 2023 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package server

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var postData []map[string]interface{}

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

func fakeMongoPost(coll string, filter primitive.M, data map[string]interface{}) bool {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true
}

func fakeMongoGetOne(value map[string]interface{}) func(string, primitive.M) map[string]interface{} {
	return func(_ string, _ primitive.M) map[string]interface{} {
		return value
	}
}

func Test_handleDeviceGroupPost(t *testing.T) {
	var deviceGroups = []configmodels.DeviceGroups{deviceGroup("group1"), deviceGroup("group2"), deviceGroup("group_no_imsis"), deviceGroup("group_no_traf_class"), deviceGroup("group_no_qos"), configmodels.DeviceGroups{}}
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
		RestfulAPIPost = fakeMongoPost
		RestfulAPIGetOne = fakeMongoGetOne(nil)

		handleDeviceGroupPost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.devGroupData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"DeviceGroupName": testGroup.DeviceGroupName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultGroup configmodels.DeviceGroups
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		json.Unmarshal(mapToByte(result), &resultGroup)
		if !reflect.DeepEqual(resultGroup, testGroup) {
			t.Errorf("Expected group %v, got %v", testGroup, resultGroup)
		}
		receivedConfigMsg, _ := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if receivedConfigMsg.PrevDevGroup.DeviceGroupName != "" {
			t.Errorf("Expected previous device group name to be empty, got %v", receivedConfigMsg.PrevDevGroup.DeviceGroupName)
		}
	}
}

func Test_handleDeviceGroupPost_alreadyExists(t *testing.T) {
	var deviceGroups = []configmodels.DeviceGroups{deviceGroup("group1"), deviceGroup("group2"), deviceGroup("group_no_imsis"), deviceGroup("group_no_traf_class"), deviceGroup("group_no_qos"), configmodels.DeviceGroups{}}
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
		var previousGroupBson bson.M
		previousGroup, _ := json.Marshal(testGroup)
		json.Unmarshal(previousGroup, &previousGroupBson)
		RestfulAPIPost = fakeMongoPost
		RestfulAPIGetOne = fakeMongoGetOne(previousGroupBson)

		handleDeviceGroupPost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.devGroupData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"DeviceGroupName": testGroup.DeviceGroupName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultGroup configmodels.DeviceGroups
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		json.Unmarshal(mapToByte(result), &resultGroup)
		if !reflect.DeepEqual(resultGroup, testGroup) {
			t.Errorf("Expected group %v, got %v", testGroup, resultGroup)
		}
		receivedConfigMsg, _ := <-subsUpdateChan
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
	var networkSlices = []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"), networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups"), configmodels.Slice{}}
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
		RestfulAPIPost = fakeMongoPost
		RestfulAPIGetOne = fakeMongoGetOne(nil)

		handleNetworkSlicePost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.sliceData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"SliceName": testSlice.SliceName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultSlice configmodels.Slice
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		json.Unmarshal(mapToByte(result), &resultSlice)
		if !reflect.DeepEqual(resultSlice, testSlice) {
			t.Errorf("Expected slice %v, got %v", testSlice, resultSlice)
		}
		receivedConfigMsg, _ := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if receivedConfigMsg.PrevSlice.SliceName != "" {
			t.Errorf("Expected previous network slice name to be empty, got %v", receivedConfigMsg.PrevSlice.SliceName)
		}
	}
}

func Test_handleNetworkSlicePost_alreadyExists(t *testing.T) {
	var networkSlices = []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"), networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups"), configmodels.Slice{}}
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
		previousSlice, _ := json.Marshal(testSlice)
		json.Unmarshal(previousSlice, &previousSliceBson)
		RestfulAPIPost = fakeMongoPost
		RestfulAPIGetOne = fakeMongoGetOne(previousSliceBson)

		handleNetworkSlicePost(&configMsg, subsUpdateChan)

		expected_collection := "webconsoleData.snapshots.sliceData"
		if postData[0]["coll"] != expected_collection {
			t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
		}
		expected_filter := bson.M{"SliceName": testSlice.SliceName}
		if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
			t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
		}
		var resultSlice configmodels.Slice
		var result map[string]interface{} = postData[0]["data"].(map[string]interface{})
		json.Unmarshal(mapToByte(result), &resultSlice)
		if !reflect.DeepEqual(resultSlice, testSlice) {
			t.Errorf("Expected slice %v, got %v", testSlice, resultSlice)
		}
		receivedConfigMsg, _ := <-subsUpdateChan
		if !reflect.DeepEqual(receivedConfigMsg.Msg, &configMsg) {
			t.Errorf("Expected config message %v, got %v", configMsg, receivedConfigMsg.Msg)
		}
		if !reflect.DeepEqual(receivedConfigMsg.PrevSlice, &testSlice) {
			t.Errorf("Expected previous network slice to be %v, got %v", testSlice, receivedConfigMsg.PrevSlice)
		}
	}
}
