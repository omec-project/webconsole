// SPDX-FileCopyrightText: 2025 Canonical Ltd
// SPDX-FileCopyrightText: 2023 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

// error case
// delete api

type DeviceGroupMockDBClient struct {
	dbadapter.DBInterface
	deviceGroups []configmodels.DeviceGroups
	err          error
}

func (db *DeviceGroupMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	if len(db.deviceGroups) == 0 {
		return nil, nil
	}
	dg := configmodels.ToBsonM(db.deviceGroups[0])
	if dg == nil {
		panic("failed to convert device group to BsonM")
	}
	return dg, nil
}

func (db *DeviceGroupMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, deviceGroup := range db.deviceGroups {
		dg := configmodels.ToBsonM(deviceGroup)
		if dg == nil {
			panic("failed to convert device groups to BsonM")
		}
		results = append(results, dg)
	}
	return results, db.err
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
		DeviceGroupName:  name,
		Imsis:            []string{"1234", "5678"},
		SiteInfo:         "demo",
		IpDomainName:     "pool1",
		IpDomainExpanded: ipdomain,
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
				deviceGroups: tc.configuredDeviceGroups,
			}
			GetDeviceGroups(c)
			resp := w.Result()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected StatusCode %d, got %d", tc.expectedCode, resp.StatusCode)
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
				t.Errorf("Expected %+v, got %+v", expected, actual)
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
		deviceGroups: []configmodels.DeviceGroups{},
	}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(bodyBytes) != "null" {
		t.Errorf("Expected body 'null', got: %v", string(bodyBytes))
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
		t.Errorf("Expected StatusCode %d, got %d", http.StatusInternalServerError, resp.StatusCode)
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
		t.Errorf("Expected response body %v, got %v", expected, actual)
	}
}

func TestGetDeviceGroupByName_DeviceGroupExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		deviceGroups: []configmodels.DeviceGroups{deviceGroup("group1")},
	}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
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
		t.Errorf("Expected %+v, got %+v", expected, actual)
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

type NetworkSliceMockDBClient struct {
	dbadapter.DBInterface
	slices []configmodels.Slice
	err    error
}

func (db *NetworkSliceMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	if len(db.slices) == 0 {
		return nil, nil
	}
	ns := configmodels.ToBsonM(db.slices[0])
	if ns == nil {
		panic("failed to convert network slice to BsonM")
	}
	return ns, nil
}

func (db *NetworkSliceMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, s := range db.slices {
		ns := configmodels.ToBsonM(s)
		if ns == nil {
			panic("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, db.err
}

func TestGetNetworkSlices(t *testing.T) {
	tests := []struct {
		name             string
		configuredSlices []configmodels.Slice
		expectedCode     int
		expectedResult   []string
	}{
		{
			name:             "No network slices return empty list",
			configuredSlices: []configmodels.Slice{},
			expectedCode:     http.StatusOK,
			expectedResult:   []string{},
		},
		{
			name: "One network slice returns a list with one name",
			configuredSlices: []configmodels.Slice{
				networkSlice("slice1"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{"slice1"},
		},
		{
			name: "Many slices returns a list with many slices names",
			configuredSlices: []configmodels.Slice{
				networkSlice("slice1"),
				networkSlice("slice2"),
				networkSlice("slice3"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{"slice1", "slice2", "slice3"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()

			dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
				slices: tc.configuredSlices,
			}
			GetNetworkSlices(c)
			resp := w.Result()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected StatusCode %d, got %d", tc.expectedCode, resp.StatusCode)
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
				t.Errorf("Expected %+v, got %+v", expected, actual)
			}
		})
	}
}

func TestGetNetworkSliceByName_NetworkSliceDoesNotExist(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		slices: []configmodels.Slice{},
	}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(bodyBytes) != "null" {
		t.Errorf("Expected body 'null', got: %v", string(bodyBytes))
	}
}

func TestGetNetworkSliceByName_DBError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		err: fmt.Errorf("mock error"),
	}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var actual map[string]string
	if err := json.Unmarshal(bodyBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expected := map[string]string{"error": "failed to retrieve network slice"}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected response body %v, got %v", expected, actual)
	}
}

func TestGetNetworkSliceByName_NetworkSliceExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		slices: []configmodels.Slice{networkSlice("slice1")},
	}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var actual configmodels.Slice
	if err := json.Unmarshal(bodyBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	expected := networkSlice("slice1")
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %+v, got %+v", expected, actual)
	}
}
