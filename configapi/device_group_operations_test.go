// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeviceGroupMockDBClient struct {
	dbadapter.DBInterface
	configuredDeviceGroups []configmodels.DeviceGroups
	postData               []map[string]any
	deleteData             []map[string]any
	err                    error
}

func (db *DeviceGroupMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	if len(db.configuredDeviceGroups) == 0 {
		return nil, nil
	}
	dg := configmodels.ToBsonM(db.configuredDeviceGroups[0])
	if dg == nil {
		logger.DbLog.Fatalln("failed to convert device group to BsonM")
	}
	return dg, nil
}

func (db *DeviceGroupMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, deviceGroup := range db.configuredDeviceGroups {
		dg := configmodels.ToBsonM(deviceGroup)
		if dg == nil {
			logger.DbLog.Fatalln("failed to convert device groups to BsonM")
		}
		results = append(results, dg)
	}
	return results, db.err
}

func (db *DeviceGroupMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	params := map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	}
	db.postData = append(db.postData, params)
	return true, nil
}

func (db *DeviceGroupMockDBClient) RestfulAPIDeleteOne(coll string, filter primitive.M) error {
	params := map[string]any{
		"coll":   coll,
		"filter": filter,
	}
	db.deleteData = append(db.deleteData, params)
	return nil
}

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
		DeviceGroupName: name,
		Imsis:           []string{"1234", "5678"},
		SiteInfo:        "demo",
		IpDomainName:    "pool1",
		IpDomainExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			ipdomain,
		},
	}
	return deviceGroup
}

func TestGetDeviceGroups(t *testing.T) {
	tests := []struct {
		name                   string
		configuredDeviceGroups []configmodels.DeviceGroups
		expectedCode           int
		expectedResult         []string
	}{
		{
			name:                   "No device groups return empty list",
			configuredDeviceGroups: []configmodels.DeviceGroups{},
			expectedCode:           http.StatusOK,
			expectedResult:         []string{},
		},
		{
			name: "One device group returns a list with one name",
			configuredDeviceGroups: []configmodels.DeviceGroups{
				deviceGroup("group1"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{"group1"},
		},
		{
			name: "Many device groups returns a list with many names",
			configuredDeviceGroups: []configmodels.DeviceGroups{
				deviceGroup("group1"),
				deviceGroup("group2"),
				deviceGroup("group3"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{"group1", "group2", "group3"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
				configuredDeviceGroups: tc.configuredDeviceGroups,
			}
			GetDeviceGroups(c)
			resp := w.Result()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("expected StatusCode %d, got %d", tc.expectedCode, resp.StatusCode)
			}
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual []string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}

			expected := tc.expectedResult
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected %+v, got %+v", expected, actual)
			}
		})
	}
}

func TestGetDeviceGroupByName_DeviceGroupDoesNotExist(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		configuredDeviceGroups: []configmodels.DeviceGroups{},
	}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusCode %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(bodyBytes) != "null" {
		t.Errorf("expected body 'null', got: %v", string(bodyBytes))
	}
}

func TestGetDeviceGroupByName_DBError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		err: fmt.Errorf("mock error"),
	}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var actual map[string]string
	if err := json.Unmarshal(bodyBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expected := map[string]string{"error": "failed to retrieve device group"}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected response body %v, got %v", expected, actual)
	}
}

func TestGetDeviceGroupByName_DeviceGroupExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		configuredDeviceGroups: []configmodels.DeviceGroups{deviceGroup("group1")},
	}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var actual configmodels.DeviceGroups
	if err := json.Unmarshal(bodyBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	expected := deviceGroup("group1")
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func Test_handleDeviceGroupPost(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{
		deviceGroup("group1"),
		deviceGroup("group2"),
		deviceGroup("group_no_imsis"),
		deviceGroup("group_no_traf_class"),
		deviceGroup("group_no_qos"),
	}
	deviceGroups[2].Imsis = []string{}
	if len(deviceGroups[3].IpDomainExpanded) > 0 {
		deviceGroups[3].IpDomainExpanded[0].UeDnnQos.TrafficClass = nil
	}
	if len(deviceGroups[4].IpDomainExpanded) > 0 {
		deviceGroups[4].IpDomainExpanded[0].UeDnnQos = nil
	}

	for _, testGroup := range deviceGroups {
		dg := testGroup

		t.Run(dg.DeviceGroupName, func(t *testing.T) {
			mockDB := &DeviceGroupMockDBClient{}
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			dbadapter.CommonDBClient = mockDB

			statusCode, err := handleDeviceGroupPost(&dg, nil)
			if err != nil {
				t.Fatalf("Could not handle device group post: %+v status code: %d", err, statusCode)
			}

			if len(mockDB.postData) == 0 {
				t.Fatal("No post operation was recorded")
			}

			if mockDB.postData[0]["coll"] != devGroupDataColl {
				t.Errorf("expected collection %v, got %v", devGroupDataColl, mockDB.postData[0]["coll"])
			}

			expectedFilter := bson.M{"group-name": dg.DeviceGroupName}
			if !reflect.DeepEqual(mockDB.postData[0]["filter"], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mockDB.postData[0]["filter"])
			}

			result := mockDB.postData[0]["data"].(map[string]any)
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("could not marshal result data: %v", err)
			}
			var resultGroup configmodels.DeviceGroups
			if err := json.Unmarshal(bytes, &resultGroup); err != nil {
				t.Fatalf("could not unmarshal result data: %v", err)
			}
			if !reflect.DeepEqual(resultGroup, dg) {
				t.Errorf("expected group %v, got %v", dg, resultGroup)
			}
		})
	}
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
	if len(deviceGroups[3].IpDomainExpanded) > 0 {
		deviceGroups[3].IpDomainExpanded[0].UeDnnQos.TrafficClass = nil
	}
	if len(deviceGroups[4].IpDomainExpanded) > 0 {
		deviceGroups[4].IpDomainExpanded[0].UeDnnQos = nil
	}

	for _, testGroup := range deviceGroups {
		dg := testGroup

		t.Run(dg.DeviceGroupName, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			mock := &DeviceGroupMockDBClient{configuredDeviceGroups: []configmodels.DeviceGroups{dg}}
			dbadapter.CommonDBClient = mock

			statusCode, err := handleDeviceGroupPost(&dg, &dg)
			if err != nil {
				t.Fatalf("handleDeviceGroupPost returned error: %+v statusCode: %d", err, statusCode)
			}

			if len(mock.postData) == 0 {
				t.Fatal("no post operation was recorded")
			}

			if mock.postData[0]["coll"] != devGroupDataColl {
				t.Errorf("expected collection %v, got %v", devGroupDataColl, mock.postData[0]["coll"])
			}

			expectedFilter := bson.M{"group-name": dg.DeviceGroupName}
			if !reflect.DeepEqual(mock.postData[0]["filter"], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mock.postData[0]["filter"])
			}

			result := mock.postData[0]["data"].(map[string]any)
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("could not marshal result map: %v", err)
			}
			var resultGroup configmodels.DeviceGroups
			if err := json.Unmarshal(bytes, &resultGroup); err != nil {
				t.Fatalf("could not unmarshal result: %v", err)
			}
			if !reflect.DeepEqual(resultGroup, dg) {
				t.Errorf("expected group %v, got %v", dg, resultGroup)
			}
		})
	}
}

func Test_handleDeviceGroupDelete(t *testing.T) {
	originalDBClient := dbadapter.CommonDBClient
	defer func() {
		dbadapter.CommonDBClient = originalDBClient
	}()
	dbClientMock := &DeviceGroupMockDBClient{}
	dbadapter.CommonDBClient = dbClientMock

	err := handleDeviceGroupDelete("group1")
	if err != nil {
		t.Fatalf("handleDeviceGroupDelete failed: %v", err)
	}

	if len(dbClientMock.deleteData) == 0 {
		t.Fatal("no delete operation was recorded")
	}

	expectedColl := devGroupDataColl
	if dbClientMock.deleteData[0]["coll"] != expectedColl {
		t.Errorf("expected collection %v, got %v", expectedColl, dbClientMock.deleteData[0]["coll"])
	}

	expectedFilter := bson.M{"group-name": "group1"}
	if !reflect.DeepEqual(dbClientMock.deleteData[0]["filter"], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, dbClientMock.deleteData[0]["filter"])
	}
}

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
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			if tc.expectedCode == http.StatusOK {
				dbadapter.CommonDBClient = &DeviceGroupMockDBClient{}
			}
			newDeviceGroup := deviceGroup("name")
			jsonBody, err := json.Marshal(newDeviceGroup)
			if err != nil {
				t.Fatalf("failed to marshal device group %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, tc.route, bytes.NewReader(jsonBody))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
		})
	}
}
