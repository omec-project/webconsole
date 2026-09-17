// Copyright (c) 2026 Intel Corporation
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	testSliceName   = "slice1"
	bitrateUnitMbps = "Mbps"
	bitrateUnitGbps = "Gbps"
)

var execCommandTimesCalled = 0

func networkSlice(name string) configmodels.Slice {
	return networkSliceWithGnbParams(name, "demo-gnb1", 1)
}

func networkSliceWithGnbParams(name string, gnbName string, gnbTac int32) configmodels.Slice {
	upf := make(map[string]any, 0)
	upf["upf-name"] = "upf"
	upf["upf-port"] = "8805"
	plmn := configmodels.SliceSiteInfoPlmn{
		Mcc: "208",
		Mnc: "93",
	}
	gnodeb := configmodels.SliceSiteInfoGNodeBs{
		Name: gnbName,
		Tac:  gnbTac,
	}
	slice_id := configmodels.SliceSliceId{
		Sst: "1",
		Sd:  "010203",
	}
	site_info := configmodels.SliceSiteInfo{
		SiteName: demoSiteName,
		Plmn:     plmn,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{gnodeb},
		Upf:      upf,
	}
	slice := configmodels.Slice{
		SliceName:       name,
		SliceId:         slice_id,
		SiteDeviceGroup: []string{testGroupName, "group2"},
		SiteInfo:        site_info,
	}
	return slice
}

type NetworkSliceMockDBClient struct {
	dbadapter.DBInterface
	slices   []configmodels.Slice
	postData []map[string]any
	putData  []map[string]any
	err      error
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
		logger.DbLog.Fatalln("failed to convert network slice to BsonM")
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
			logger.DbLog.Fatalln("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, db.err
}

func (db *NetworkSliceMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	params := map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	}
	db.postData = append(db.postData, params)
	return true, nil
}

func (db *NetworkSliceMockDBClient) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error) {
	params := map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   putData,
	}
	db.putData = append(db.putData, params)
	return true, db.err
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
				networkSlice(testSliceName),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{testSliceName},
		},
		{
			name: "Many slices returns a list with many slices names",
			configuredSlices: []configmodels.Slice{
				networkSlice(testSliceName),
				networkSlice("slice2"),
				networkSlice("slice3"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{testSliceName, "slice2", "slice3"},
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

func TestGetNetworkSliceByName_NetworkSliceDoesNotExist(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		slices: []configmodels.Slice{},
	}
	c.Params = append(c.Params, gin.Param{Key: sliceNameKey, Value: testSliceName})
	GetNetworkSliceByName(c)
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

func TestGetNetworkSliceByName_DBError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		err: fmt.Errorf("mock error"),
	}
	c.Params = append(c.Params, gin.Param{Key: sliceNameKey, Value: testSliceName})
	GetNetworkSliceByName(c)
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

	expected := map[string]string{errorKey: errMsgRetrieveNetworkSlice}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected response body %v, got %v", expected, actual)
	}
}

func TestGetNetworkSliceByName_NetworkSliceExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{
		slices: []configmodels.Slice{networkSlice(testSliceName)},
	}
	c.Params = append(c.Params, gin.Param{Key: sliceNameKey, Value: testSliceName})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var actual configmodels.Slice
	if err := json.Unmarshal(bodyBytes, &actual); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	expected := networkSlice(testSliceName)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", "YOUR COMMAND"}
	cs = append(cs, args...)
	cmd := exec.CommandContext(context.Background(), os.Args[0], cs...)
	execCommandTimesCalled += 1
	return cmd
}

func Test_sendPebbleNotification_on_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	origSync := syncSubscribersOnSliceCreateOrUpdate
	syncSubscribersOnSliceCreateOrUpdate = func(_, _ configmodels.Slice) (int, error) {
		return http.StatusOK, nil
	}
	defer func() { syncSubscribersOnSliceCreateOrUpdate = origSync }()

	numPebbleNotificationsSent := execCommandTimesCalled

	slice := networkSlice(testSliceName)
	prevSlice := configmodels.Slice{}

	factory.WebUIConfig.Configuration.SendPebbleNotifications = true
	originalDBClient := dbadapter.CommonDBClient
	defer func() {
		dbadapter.CommonDBClient = originalDBClient
	}()
	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{}

	statusCode, err := handleNetworkSlicePost(slice, prevSlice)
	if err != nil {
		t.Errorf("could not handle network slice post: %+v statusCode: %d", err, statusCode)
	}
	if execCommandTimesCalled != numPebbleNotificationsSent+1 {
		t.Errorf("unexpected number of Pebble notifications: %v. Should be: %v", execCommandTimesCalled, numPebbleNotificationsSent+1)
	}
}

func Test_sendPebbleNotification_off_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	execCommandTimesCalled = 0

	origSync := syncSubscribersOnSliceCreateOrUpdate
	syncSubscribersOnSliceCreateOrUpdate = func(_, _ configmodels.Slice) (int, error) {
		return http.StatusOK, nil
	}
	defer func() { syncSubscribersOnSliceCreateOrUpdate = origSync }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SendPebbleNotifications: false,
		},
	}

	slice := configmodels.Slice{SliceName: testSliceName}
	prevSlice := configmodels.Slice{}
	originalDBClient := dbadapter.CommonDBClient
	defer func() {
		dbadapter.CommonDBClient = originalDBClient
	}()
	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{}

	statusCode, err := handleNetworkSlicePost(slice, prevSlice)
	if err != nil {
		t.Errorf("handleNetworkSlicePost returned error: %+v statusCode: %d", err, statusCode)
	}

	if execCommandTimesCalled != 0 {
		t.Errorf("expected 0 Pebble notifications, but got %v", execCommandTimesCalled)
	}
}

func Test_handleNetworkSlicePost(t *testing.T) {
	networkSlices := []configmodels.Slice{
		networkSlice(testSliceName),
		networkSlice("slice2"),
		networkSlice("slice_no_gnodeb"),
		networkSlice("slice_no_device_groups"),
	}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}

	for _, testSlice := range networkSlices {
		ts := testSlice

		t.Run(ts.SliceName, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() {
				dbadapter.CommonDBClient = originalDBClient
			}()
			mock := &NetworkSliceMockDBClient{slices: []configmodels.Slice{ts}}
			dbadapter.CommonDBClient = mock

			statusCode, err := handleNetworkSlicePost(ts, ts)
			if err != nil {
				t.Fatalf("handleNetworkSlicePost returned error: %+v status code: %d", err, statusCode)
			}

			if len(mock.postData) == 0 {
				t.Fatal("expected a post operation but none was recorded")
			}

			if mock.postData[0][collKey] != sliceDataColl {
				t.Errorf("expected collection %v, got %v", sliceDataColl, mock.postData[0][collKey])
			}

			expectedFilter := bson.M{sliceNameKey: ts.SliceName}
			if !reflect.DeepEqual(mock.postData[0][filterKey], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mock.postData[0][filterKey])
			}

			result := mock.postData[0][dataKey].(map[string]any)
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result data: %v", err)
			}
			var resultSlice configmodels.Slice
			if err := json.Unmarshal(bytes, &resultSlice); err != nil {
				t.Fatalf("Failed to unmarshal result data: %v", err)
			}
			if !reflect.DeepEqual(resultSlice, ts) {
				t.Errorf("expected slice %v, got %v", ts, resultSlice)
			}
		})
	}
}

func TestNetworkSlicePostHandler_NetworkSliceNameValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		expectedCode int
	}{
		{
			name:         "Network Slice invalid name (invalid token)",
			route:        "/config/v1/network-slice/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Network Slice invalid name (invalid length)",
			route:        "/config/v1/network-slice/" + genLongString(257),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Network Slice valid name",
			route:        "/config/v1/network-slice/slice1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			if tc.expectedCode == http.StatusOK {
				dbadapter.CommonDBClient = &NetworkSliceMockDBClient{}
			}
			jsonBody, err := json.Marshal(networkSlice("name"))
			if err != nil {
				t.Fatalf("failed to marshal device group %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, tc.route, bytes.NewReader(jsonBody))
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

// SlicePlmnValidationMockDBClient answers slice and device group lookups by collection, unlike
// NetworkSliceMockDBClient which always returns the same stored slice regardless of collection.
type SlicePlmnValidationMockDBClient struct {
	dbadapter.DBInterface
	deviceGroups map[string]configmodels.DeviceGroups
	postData     []map[string]any
}

func (db *SlicePlmnValidationMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if coll != devGroupDataColl {
		return nil, nil
	}
	name, _ := filter[groupNameKey].(string)
	dg, ok := db.deviceGroups[name]
	if !ok {
		return nil, nil
	}
	return configmodels.ToBsonM(dg), nil
}

func (db *SlicePlmnValidationMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.postData = append(db.postData, map[string]any{collKey: collName, filterKey: filter, dataKey: postData})
	return true, nil
}

// Subscriber DB records are keyed by (imsi, PLMN). Accepting a PLMN change on an existing slice
// would leave the old records in place and post new ones under the new PLMN, duplicating every
// subscriber in the slice's device groups instead of moving them
func TestNetworkSlicePostHandler_RejectsPlmnChangeOnExistingSlice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	existing := networkSlice(testSliceName)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	mock := &NetworkSliceMockDBClient{slices: []configmodels.Slice{existing}}
	dbadapter.CommonDBClient = mock

	updated := networkSlice(testSliceName)
	updated.SiteInfo.Plmn.Mcc = "123"
	updated.SiteInfo.Plmn.Mnc = "45"
	jsonBody, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("failed to marshal network slice: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected `%d`, got `%d`: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	if len(mock.postData) != 0 {
		t.Errorf("expected the PLMN change to be rejected before any write, got %d posted documents", len(mock.postData))
	}
}

// A slice that never had device groups attached could not have synced a subscriber under its
// zero-value PLMN, since syncSubscribersOnSliceCreateOrUpdate only ever provisions device groups
// attached to the slice, so assigning a real PLMN in that case is safe.
func TestNetworkSlicePostHandler_AllowsPlmnAssignmentWhenSliceHadNoDeviceGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	existing := networkSlice(testSliceName)
	existing.SiteInfo.Plmn = configmodels.SliceSiteInfoPlmn{}
	existing.SiteDeviceGroup = nil
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	mock := &NetworkSliceMockDBClient{slices: []configmodels.Slice{existing}}
	dbadapter.CommonDBClient = mock

	updated := networkSlice(testSliceName)
	updated.SiteDeviceGroup = nil
	jsonBody, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("failed to marshal network slice: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected `%d`, got `%d`: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// A zero-value PLMN does not mean no subscriber was ever synced under it:
// syncSubscribersOnSliceCreateOrUpdate uses mcc+mnc as the serving PLMN key even when both are
// empty, so a slice that already had device groups attached can already have subscriber records
// filed under that empty key. Assigning a real PLMN must still be rejected in that case, same as
// any other PLMN change, to avoid duplicating those records under the new PLMN.
func TestNetworkSlicePostHandler_RejectsPlmnAssignmentWhenSliceHadDeviceGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	existing := networkSlice(testSliceName)
	existing.SiteInfo.Plmn = configmodels.SliceSiteInfoPlmn{}
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	mock := &NetworkSliceMockDBClient{slices: []configmodels.Slice{existing}}
	dbadapter.CommonDBClient = mock

	updated := networkSlice(testSliceName)
	jsonBody, err := json.Marshal(updated)
	if err != nil {
		t.Fatalf("failed to marshal network slice: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected `%d`, got `%d`: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	if len(mock.postData) != 0 {
		t.Errorf("expected the PLMN assignment to be rejected before any write, got %d posted documents", len(mock.postData))
	}
}

// The first digits of an IMSI are its home PLMN, so a device group carrying a subscriber from a
// different PLMN must not be attached to a slice - it would file that subscriber's records under
// a serving PLMN it does not belong to.
func TestNetworkSlicePostHandler_RejectsDeviceGroupImsiNotMatchingPlmn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	mock := &SlicePlmnValidationMockDBClient{
		deviceGroups: map[string]configmodels.DeviceGroups{
			testGroupName: {
				DeviceGroupName: testGroupName,
				Imsis:           []string{"999990000000001"},
			},
		},
	}
	dbadapter.CommonDBClient = mock

	slice := networkSlice(testSliceName)
	slice.SiteDeviceGroup = []string{testGroupName}
	jsonBody, err := json.Marshal(slice)
	if err != nil {
		t.Fatalf("failed to marshal network slice: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected `%d`, got `%d`: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "999990000000001") {
		t.Errorf("expected error to mention the mismatched IMSI, got `%s`", w.Body.String())
	}
	if len(mock.postData) != 0 {
		t.Errorf("expected the mismatch to be rejected before any write, got %d posted documents", len(mock.postData))
	}
}

func TestNetworkSlicePostHandler_NetworkSliceGnbTacValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name          string
		route         string
		inputData     configmodels.Slice
		expectedCode  int
		expectedError string
	}{
		{
			name:          "Network Slice invalid gNB name",
			route:         "/config/v1/network-slice/slice-1",
			inputData:     networkSliceWithGnbParams("slice-1", "", 3),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid gNB name",
		},
		{
			name:          "Network Slice invalid gNB TAC",
			route:         "/config/v1/network-slice/slice-1",
			inputData:     networkSliceWithGnbParams("slice-1", "valid-gnb", 0),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid TAC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tc.inputData)
			if err != nil {
				t.Fatalf("failed to marshal device group %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, tc.route, bytes.NewReader(jsonBody))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("expected body to contain error about  `%v`, got `%v`", tc.expectedError, w.Body.String())
			}
		})
	}
}

func TestAggregateQoS_SumsCorrectly(t *testing.T) {
	qosList := []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		{
			DnnMbrUplink:   100,
			DnnMbrDownlink: 200,
			BitrateUnit:    bitrateUnitMbps,
			TrafficClass:   &configmodels.TrafficClassInfo{Qci: 9},
		},
		{
			DnnMbrUplink:   50,
			DnnMbrDownlink: 75,
			BitrateUnit:    bitrateUnitMbps,
			TrafficClass:   &configmodels.TrafficClassInfo{Qci: 5},
		},
	}

	result := aggregateQoS(qosList)

	if result.DnnMbrUplink != 150 {
		t.Fatalf("expected UL 150, got %d", result.DnnMbrUplink)
	}
	if result.DnnMbrDownlink != 275 {
		t.Fatalf("expected DL 275, got %d", result.DnnMbrDownlink)
	}

	if result.TrafficClass == nil || result.TrafficClass.Qci != 5 {
		t.Fatalf("expected lowest QCI to win (5), got %v", result.TrafficClass)
	}

	if result.BitrateUnit != bitrateUnitMbps {
		t.Fatalf("expected BitrateUnit Mbps, got %s", result.BitrateUnit)
	}
}

func TestAggregateQoS_MixedUnits(t *testing.T) {
	qosList := []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		{DnnMbrUplink: 10, BitrateUnit: ""},
		{DnnMbrUplink: 20, BitrateUnit: "Kbps"},
		{DnnMbrUplink: 30, BitrateUnit: ""},
	}

	result := aggregateQoS(qosList)
	if result.BitrateUnit != "Kbps" {
		t.Fatalf("expected first non-empty unit 'Kbps', got %s", result.BitrateUnit)
	}
	if result.DnnMbrUplink != 60 {
		t.Fatalf("expected UL 60, got %d", result.DnnMbrUplink)
	}
}

// The aggregate is served the same way a single rate is, so it has the same ceiling, and it is the
// one place a sum can exceed what each rate was validated against on its own. It can also be
// reached from below: a group written before the rates were bounded holds the math.MaxInt64 the old
// ingest path clamped a negative rate to, and adding two of those plainly gives -2, which
// ConvertToString serves as "18446744073709551614 bps".
func TestAggregateQoS_SaturatesInsteadOfWrapping(t *testing.T) {
	testCases := []struct {
		name     string
		rates    []int64
		expected int64
	}{
		{"two rates that each fit but do not together", []int64{40000 * GBPS, 40000 * GBPS}, maxDeviceGroupBitrateBps},
		{"two device groups written before the rates were bounded", []int64{math.MaxInt64, math.MaxInt64}, maxDeviceGroupBitrateBps},
		{"rates that fit are left alone", []int64{10 * GBPS, 20 * GBPS}, 30 * GBPS},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var qosList []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos
			for _, rate := range tc.rates {
				qosList = append(qosList, configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
					DnnMbrUplink:   rate,
					DnnMbrDownlink: rate,
					BitrateUnit:    bitrateUnitBps,
				})
			}
			result := aggregateQoS(qosList)
			if result.DnnMbrUplink != tc.expected || result.DnnMbrDownlink != tc.expected {
				t.Errorf("aggregate = %d/%d, want %d", result.DnnMbrUplink, result.DnnMbrDownlink, tc.expected)
			}
			// The point of the bound is the string, so assert on that rather than only the number:
			// a rate past it is rendered in bps, which nas's Session-AMBR converter reads as "unit
			// not used" with a numeral that does not parse.
			if served := ConvertToString(uint64(result.DnnMbrUplink)); strings.HasSuffix(served, " bps") {
				t.Errorf("the aggregate is served as %q, which no consumer can read", served)
			}
		})
	}
}

func TestAggregateQoS_EmptyList(t *testing.T) {
	result := aggregateQoS(nil)
	if result.DnnMbrUplink != 0 || result.DnnMbrDownlink != 0 || result.BitrateUnit != "" || result.TrafficClass != nil {
		t.Fatalf("expected zero value for empty list, got %+v", result)
	}
}

func TestBuildSmProvisionedDataDocument(t *testing.T) {
	snssai := &models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}
	dnnMap := map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		dnnInternet: {
			{
				DnnMbrUplink:   2000000,
				DnnMbrDownlink: 5000000,
				TrafficClass: &configmodels.TrafficClassInfo{
					Qci: 9,
				},
			},
		},
	}

	doc, err := buildSmProvisionedDataDocument(snssai, dnnMap, "208", "93", "208930100007487")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := doc[ueIdKey]; got != testSubscriberImsi {
		t.Fatalf("unexpected ueId: %v", got)
	}

	singleNssai, ok := doc["singlenssai"].(map[string]interface{})
	if !ok {
		t.Fatalf("singlenssai has unexpected type: %T", doc["singlenssai"])
	}
	if singleNssai["sd"] != "010203" {
		t.Fatalf("unexpected sd: %v", singleNssai["sd"])
	}

	dnnConfigurations, ok := doc["dnnconfigurations"].(map[string]interface{})
	if !ok {
		t.Fatalf("dnnconfigurations has unexpected type: %T", doc["dnnconfigurations"])
	}
	internet, ok := dnnConfigurations[dnnInternet].(map[string]interface{})
	if !ok {
		t.Fatalf("internet dnn config has unexpected type: %T", dnnConfigurations[dnnInternet])
	}

	qos, ok := internet["5gQosProfile"].(map[string]interface{})
	if !ok {
		t.Fatalf("5gQosProfile has unexpected type: %T", internet["5gQosProfile"])
	}
	arp, ok := qos["arp"].(map[string]interface{})
	if !ok {
		t.Fatalf("arp has unexpected type: %T", qos["arp"])
	}
	if arp["preemptCap"] != models.PREEMPTIONCAPABILITY_NOT_PREEMPT {
		t.Fatalf("unexpected preemptCap: %v", arp["preemptCap"])
	}
	if arp[priorityLevelKey] != int32(8) {
		t.Fatalf("unexpected arp priorityLevel: %v", arp[priorityLevelKey])
	}

	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal document: %v", err)
	}
	if strings.Contains(string(encoded), "{}") {
		t.Fatalf("document should not contain empty objects: %s", encoded)
	}
}

func TestUpdateSmProvisionedData_UsesPutOne(t *testing.T) {
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	mock := &NetworkSliceMockDBClient{}
	dbadapter.CommonDBClient = mock

	snssai := &models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}
	dnnMap := map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		dnnInternet: {
			{
				DnnMbrUplink:   2000000,
				DnnMbrDownlink: 5000000,
				TrafficClass:   &configmodels.TrafficClassInfo{Qci: 9},
			},
		},
	}

	if err := updateSmProvisionedData(snssai, dnnMap, "208", "93", "208930100007487"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.postData) != 0 {
		t.Fatalf("expected no post calls, got %d", len(mock.postData))
	}
	if len(mock.putData) != 1 {
		t.Fatalf("expected one put call, got %d", len(mock.putData))
	}
	data, ok := mock.putData[0][dataKey].(map[string]any)
	if !ok {
		t.Fatalf("unexpected put payload type: %T", mock.putData[0][dataKey])
	}
	if _, ok = data["singlenssai"]; !ok {
		t.Fatal("expected singlenssai key in put payload")
	}
	if _, ok = data["dnnconfigurations"]; !ok {
		t.Fatal("expected dnnconfigurations key in put payload")
	}
}

func filteringRuleWithRates(unit string, mbrUl, mbrDl, gbrUl, gbrDl int32) configmodels.SliceApplicationFilteringRules {
	return configmodels.SliceApplicationFilteringRules{
		RuleName:       "rate-rule",
		BitrateUnit:    unit,
		AppMbrUplink:   mbrUl,
		AppMbrDownlink: mbrDl,
		AppGbrUplink:   gbrUl,
		AppGbrDownlink: gbrDl,
		TrafficClass:   &configmodels.TrafficClassInfo{Qci: 9, Arp: 1},
	}
}

// Guaranteed rates are configured in the rule's bitrate-unit, like the maximum rates, and must be
// normalised to bps on the same path. Left unconverted, a rule written in Mbps reaches the PCF a
// million times too small.
func TestNormalizeConvertsGuaranteedBitRatesToBps(t *testing.T) {
	slice := &configmodels.Slice{
		ApplicationFilteringRules: []configmodels.SliceApplicationFilteringRules{
			filteringRuleWithRates(bitrateUnitMbps, 50, 50, 10, 20),
		},
	}

	normalizeApplicationFilteringRules(slice)

	rule := slice.ApplicationFilteringRules[0]
	if rule.AppGbrUplink != 10_000_000 {
		t.Errorf("AppGbrUplink = %d, want 10000000 after normalising 10 Mbps", rule.AppGbrUplink)
	}
	if rule.AppGbrDownlink != 20_000_000 {
		t.Errorf("AppGbrDownlink = %d, want 20000000 after normalising 20 Mbps", rule.AppGbrDownlink)
	}
	if rule.AppMbrUplink != 50_000_000 {
		t.Errorf("AppMbrUplink = %d, want the maximum rates still normalised", rule.AppMbrUplink)
	}
}

// A negative rate must not become the largest storable one: an operator who configures -1 must not
// be served a 2.1 Gbps rate.
func TestConvertBitrateToInt32(t *testing.T) {
	testCases := []struct {
		name     string
		bitrate  int64
		expected int32
	}{
		{"unset", 0, 0},
		{"within the field", 10_000_000, 10_000_000},
		{"negative is not a rate", -1, 0},
		{"beyond the field is capped", math.MaxInt32 + 1, math.MaxInt32},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := convertBitrateToInt32(tc.bitrate); got != tc.expected {
				t.Errorf("convertBitrateToInt32(%d) = %d, want %d", tc.bitrate, got, tc.expected)
			}
		})
	}
}

// A rate that is negative, or too large for the field it is stored in, cannot be served as
// configured, so the slice is rejected rather than accepted with a different rate than was asked
// for.
func TestNetworkSlicePostHandler_BitrateValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name          string
		rule          configmodels.SliceApplicationFilteringRules
		expectedCode  int
		expectedError string
	}{
		{
			name:          "negative maximum bit rate",
			rule:          filteringRuleWithRates(bitrateUnitMbps, -1, 10, 0, 0),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid app-mbr-uplink",
		},
		{
			name:          "negative guaranteed bit rate",
			rule:          filteringRuleWithRates(bitrateUnitMbps, 10, 10, 0, -5),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid app-gbr-downlink",
		},
		{
			name:          "guaranteed bit rate too large to store",
			rule:          filteringRuleWithRates(bitrateUnitGbps, 2, 2, 3, 0),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid app-gbr-uplink",
		},
		{
			name:         "rates that fit are accepted",
			rule:         filteringRuleWithRates(bitrateUnitGbps, 2, 2, 1, 1),
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			// Installed for every case, not only the accepted one: without it a rejected rate that
			// slipped through would fail this test with a 500 from the nil client rather than the
			// 200 that is the defect.
			dbadapter.CommonDBClient = &NetworkSliceMockDBClient{}
			slice := networkSlice(testSliceName)
			slice.ApplicationFilteringRules = []configmodels.SliceApplicationFilteringRules{tc.rule}
			jsonBody, err := json.Marshal(slice)
			if err != nil {
				t.Fatalf("failed to marshal network slice %v", err)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if tc.expectedError != "" && !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("expected body to contain error about `%v`, got `%v`", tc.expectedError, w.Body.String())
			}
		})
	}
}

// A GET returns the stored rule, so the unit has to describe the stored value. The property that
// matters is idempotence: posting back what a GET returned must not multiply the rates again.
func TestNormalizeRewritesTheUnitToTheStoredOne(t *testing.T) {
	slice := &configmodels.Slice{
		ApplicationFilteringRules: []configmodels.SliceApplicationFilteringRules{
			filteringRuleWithRates(bitrateUnitMbps, 50, 50, 10, 20),
		},
	}

	normalizeApplicationFilteringRules(slice)

	if got := slice.ApplicationFilteringRules[0].BitrateUnit; got != bitrateUnitBps {
		t.Errorf("BitrateUnit = %q, want %q once the rates are bps", got, bitrateUnitBps)
	}

	normalizeApplicationFilteringRules(slice)

	rule := slice.ApplicationFilteringRules[0]
	if rule.AppMbrUplink != 50_000_000 || rule.AppGbrUplink != 10_000_000 {
		t.Errorf("normalising the stored rule again changed it: MBR %d, GBR %d", rule.AppMbrUplink, rule.AppGbrUplink)
	}
}

// ConvertToString renders the rate served to the PCF, so what it drops is what the network does
// not deliver. Integer division chose the largest unit and truncated to it: 2147000000 bps was
// served as "2 Gbps", 147 Mbps below the rate configured, where "2147 Mbps" describes it exactly.
//
// The table has the shape it does because of what the next hop can read, not because of what is
// true. omec-project/smf's GetBitRate has no case for bps and defaults to Mbps, so an exact
// "1500 bps" would signal the UE 1500 Mbps; and it reads the numeral into a uint16, so an exact
// "65536 Mbps" reaches the UE as 0 and a truthful "2147000 Kbps" as 49848. Both halves of that --
// the unit and the size of the numeral -- are what the cases below pin.
func TestConvertToStringNamesTheLargestUnitThatIsExactAndReadable(t *testing.T) {
	tests := []struct {
		name string
		bps  uint64
		want string
	}{
		{"a whole number of Gbps", 2000000000, "2 Gbps"},
		{"a whole number of Mbps but not of Gbps", 2147000000, "2147 Mbps"},
		{"a whole number of Mbps", 10000000, "10 Mbps"},
		{"a whole number of Kbps", 20000, "20 Kbps"},
		{"not a whole number of Kbps, truncated rather than served in bps", 1500, "1 Kbps"},
		{"no unit is exact, so the smallest that fits the numeral", 2147000001, "2147 Mbps"},
		{"the largest numeral a consumer can hold", 65535000000, "65535 Mbps"},
		{"one Mbps past that numeral, so the unit above it", 65536000000, "65 Gbps"},
		{"a whole number of Kbps whose numeral does not fit", 65536000, "65 Mbps"},
		{"below a Kbps, which has no smaller unit to fall back to", 500, "500 bps"},
		{"past what a Gbps numeral holds, which only a device-group rate reaches", 9223372036854775807, "9223372036854775807 bps"},
		{"no rate", 0, "0 bps"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ConvertToString(tc.bps); got != tc.want {
				t.Errorf("ConvertToString(%d) = %q, want %q", tc.bps, got, tc.want)
			}
		})
	}
}

// A rule written before the ingest path started storing the unit holds bps rates under whatever
// unit the operator posted. The GET has to say bps, or the operator reads a rate labelled a
// thousand times its own value -- and posting that document back unchanged multiplies it again,
// which is the way a slice's rates drift without anyone editing them.
func TestGetNetworkSliceByNameLabelsStoredRatesAsBps(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	// What a rule written before that contract looks like in the database: rates already in bps,
	// beside the unit the operator posted them in.
	// A kbps rule, not an Mbps one: 2000000 re-multiplied is 2000000000, which still fits the
	// stored field. The label is what stops it, and the assertion that catches the drift is the
	// comparison at the end rather than the status code -- an Mbps rule this size is refused by
	// the rate validation instead, which is a different failure and would hide this one.
	stored := networkSlice(testSliceName)
	stored.ApplicationFilteringRules = []configmodels.SliceApplicationFilteringRules{
		filteringRuleWithRates("kbps", 2000000, 1000000, 500000, 250000),
	}
	dbadapter.CommonDBClient = &NetworkSliceMockDBClient{slices: []configmodels.Slice{stored}}

	c.Params = append(c.Params, gin.Param{Key: sliceNameKey, Value: testSliceName})
	GetNetworkSliceByName(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected StatusCode %d, got %d", http.StatusOK, w.Code)
	}
	var returned configmodels.Slice
	if err := json.Unmarshal(w.Body.Bytes(), &returned); err != nil {
		t.Fatalf("failed to unmarshal the returned slice: %v", err)
	}
	if len(returned.ApplicationFilteringRules) != 1 {
		t.Fatalf("expected 1 filtering rule, got %d", len(returned.ApplicationFilteringRules))
	}
	rule := returned.ApplicationFilteringRules[0]
	if rule.BitrateUnit != bitrateUnitBps {
		t.Errorf("bitrate-unit = %q, want %q: the stored rates are bps", rule.BitrateUnit, bitrateUnitBps)
	}
	if rule.AppMbrUplink != 2000000 || rule.AppGbrUplink != 500000 {
		t.Errorf("the returned rates were altered: mbr-ul = %d, gbr-ul = %d", rule.AppMbrUplink, rule.AppGbrUplink)
	}

	// The property the label exists for, driven through the write path the operator's tool uses
	// rather than through the normalizer alone: validation reads the unit before anything is
	// converted, so a test that called normalize directly would keep passing if that order
	// changed.
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)
	postMock := &NetworkSliceMockDBClient{}
	dbadapter.CommonDBClient = postMock

	jsonBody, err := json.Marshal(returned)
	if err != nil {
		t.Fatalf("failed to marshal the returned slice: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/network-slice/"+testSliceName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, req)

	if postRecorder.Code != http.StatusOK {
		t.Fatalf("posting back what the GET returned was refused with %d: %s",
			postRecorder.Code, postRecorder.Body.String())
	}
	if len(postMock.postData) == 0 {
		t.Fatal("expected the slice to be stored")
	}
	var storedAgain configmodels.Slice
	if err := json.Unmarshal(configmodels.MapToByte(postMock.postData[0][dataKey].(map[string]any)), &storedAgain); err != nil {
		t.Fatalf("failed to unmarshal the stored slice: %v", err)
	}
	if !reflect.DeepEqual(storedAgain.ApplicationFilteringRules, returned.ApplicationFilteringRules) {
		t.Errorf("posting back what a GET returned changed the rule: %+v was stored as %+v",
			returned.ApplicationFilteringRules, storedAgain.ApplicationFilteringRules)
	}
}
