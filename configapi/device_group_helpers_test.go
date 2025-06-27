// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockMongoDGPost struct {
	dbadapter.DBInterface
}

func (m *MockMongoDGPost) RestfulAPIPost(coll string, filter primitive.M, data map[string]interface{}) (bool, error) {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true, nil
}

func (m *MockMongoDGPost) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

type MockMongoDeviceGroupGetOne struct {
	dbadapter.DBInterface
	testGroup configmodels.DeviceGroups
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

func Test_handleDeviceGroupPost(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{
		deviceGroup("group1"), deviceGroup("group2"),
		deviceGroup("group_no_imsis"), deviceGroup("group_no_traf_class"), deviceGroup("group_no_qos"),
	}
	deviceGroups[2].Imsis = []string{}
	deviceGroups[3].IpDomainExpanded.UeDnnQos.TrafficClass = nil
	deviceGroups[4].IpDomainExpanded.UeDnnQos = nil
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testGroup := range deviceGroups {
		dg := testGroup

		t.Run(dg.DeviceGroupName, func(t *testing.T) {
			postData = make([]map[string]interface{}, 0)
			mockDB := &MockMongoDGPost{}
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			dbadapter.CommonDBClient = mockDB

			statusCode, err := handleDeviceGroupPost(&dg, nil)
			if err != nil {
				t.Fatalf("Could not handle device group post: %v status code: %v", err, statusCode)
			}

			if len(postData) == 0 {
				t.Fatal("No post operation was recorded")
			}

			if postData[0]["coll"] != devGroupDataColl {
				t.Errorf("Expected collection %v, got %v", devGroupDataColl, postData[0]["coll"])
			}

			expectedFilter := bson.M{"group-name": dg.DeviceGroupName}
			if !reflect.DeepEqual(postData[0]["filter"], expectedFilter) {
				t.Errorf("Expected filter %v, got %v", expectedFilter, postData[0]["filter"])
			}

			result := postData[0]["data"].(map[string]interface{})
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Could not marshal result data: %v", err)
			}
			var resultGroup configmodels.DeviceGroups
			if err := json.Unmarshal(bytes, &resultGroup); err != nil {
				t.Fatalf("Could not unmarshal result data: %v", err)
			}
			if !reflect.DeepEqual(resultGroup, dg) {
				t.Errorf("Expected group %v, got %v", dg, resultGroup)
			}
		})
	}
}

type MockMongoDeviceGroupCombined struct {
	testGroup configmodels.DeviceGroups
	dbadapter.DBInterface
}

func (m *MockMongoDeviceGroupCombined) RestfulAPIPost(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true, nil
}

func (m *MockMongoDeviceGroupCombined) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	bytes, err := json.Marshal(m.testGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}
	return result, nil
}

func (m *MockMongoDeviceGroupCombined) Client() *mongo.Client {
	return nil
}

func (m *MockMongoDeviceGroupCombined) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func Test_handleDeviceGroupPost_alreadyExists(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{
		deviceGroup("group1"),
		deviceGroup("group2"),
		deviceGroup("group_no_imsis"),
		deviceGroup("group_no_traf_class"),
		deviceGroup("group_no_qos"),
	}
	deviceGroups[2].Imsis = []string{}
	deviceGroups[3].IpDomainExpanded.UeDnnQos.TrafficClass = nil
	deviceGroups[4].IpDomainExpanded.UeDnnQos = nil

	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testGroup := range deviceGroups {
		dg := testGroup

		t.Run(dg.DeviceGroupName, func(t *testing.T) {
			postData = make([]map[string]interface{}, 0)

			mock := &MockMongoDeviceGroupCombined{testGroup: dg}
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			dbadapter.CommonDBClient = mock

			statusCode, err := handleDeviceGroupPost(&dg, &dg)
			if err != nil {
				t.Fatalf("handleDeviceGroupPost returned error: %v statusCode: %v", err, statusCode)
			}

			if len(postData) == 0 {
				t.Fatal("No post operation was recorded")
			}

			if postData[0]["coll"] != devGroupDataColl {
				t.Errorf("Expected collection %v, got %v", devGroupDataColl, postData[0]["coll"])
			}

			expectedFilter := bson.M{"group-name": dg.DeviceGroupName}
			if !reflect.DeepEqual(postData[0]["filter"], expectedFilter) {
				t.Errorf("Expected filter %v, got %v", expectedFilter, postData[0]["filter"])
			}

			result := postData[0]["data"].(map[string]interface{})
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Could not marshal result map: %v", err)
			}
			var resultGroup configmodels.DeviceGroups
			if err := json.Unmarshal(bytes, &resultGroup); err != nil {
				t.Fatalf("Could not unmarshal result: %v", err)
			}
			if !reflect.DeepEqual(resultGroup, dg) {
				t.Errorf("Expected group %v, got %v", dg, resultGroup)
			}
		})
	}
}

func Test_handleDeviceGroupDelete(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{deviceGroup("group1")}

	for _, testGroup := range deviceGroups {
		t.Run(testGroup.DeviceGroupName, func(t *testing.T) {
			deleteData = make([]map[string]interface{}, 0)
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			dbadapter.CommonDBClient = &MockMongoDeleteOne{}

			err := handleDeviceGroupDelete(testGroup.DeviceGroupName)
			if err != nil {
				t.Fatalf("handleDeviceGroupDelete failed: %v", err)
			}

			if len(deleteData) == 0 {
				t.Fatal("No delete operation was recorded")
			}

			expectedColl := devGroupDataColl
			if deleteData[0]["coll"] != expectedColl {
				t.Errorf("Expected collection %v, got %v", expectedColl, deleteData[0]["coll"])
			}

			expectedFilter := bson.M{"group-name": testGroup.DeviceGroupName}
			if !reflect.DeepEqual(deleteData[0]["filter"], expectedFilter) {
				t.Errorf("Expected filter %v, got %v", expectedFilter, deleteData[0]["filter"])
			}
		})
	}
}

const DEVICE_GROUP_CONFIG = `{
  "group-name": "string",
  "imsis": [
    "string"
  ],
  "ip-domain-expanded": {
    "dnn": "string",
    "dns-primary": "string",
    "dns-secondary": "string",
    "mtu": 0,
    "ue-dnn-qos": {
      "bitrate-unit": "string",
      "dnn-mbr-downlink": 0,
      "dnn-mbr-uplink": 0,
      "traffic-class": {
        "arp": 0,
        "name": "string",
        "pdb": 0,
        "pelr": 0,
        "qci": 0
      }
    },
    "ue-ip-pool": "string"
  },
  "ip-domain-name": "string",
  "site-info": "string"
}`

func TestDeviceGroupPostHandler_DeviceGroupNameValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		expectedCode int
	}{
		{
			name:         "Device Group invalid name (invalid token)",
			route:        "/config/v1/device-group/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Device Group invalid name (invalid length)",
			route:        "/config/v1/device-group/" + genLongString(257),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Device Group valid name",
			route:        "/config/v1/device-group/valid-devicegroup",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 1)
			originalDBClient := dbadapter.CommonDBClient
			defer func() { configChannel = origChannel; dbadapter.CommonDBClient = originalDBClient }()
			if tc.expectedCode == http.StatusOK {
				dbadapter.CommonDBClient = &MockMongoClientEmptyDB{}
			}
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(DEVICE_GROUP_CONFIG))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
		})
	}
}
