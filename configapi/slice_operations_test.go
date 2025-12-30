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
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
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
	slices   []configmodels.Slice
	postData []map[string]any
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
		logger.AppLog.Fatalln("failed to convert network slice to BsonM")
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
			logger.AppLog.Fatalln("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, db.err
}

func (db *NetworkSliceMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	params := map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	}
	db.postData = append(db.postData, params)
	return true, nil
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
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
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
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
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

	expected := map[string]string{"error": "failed to retrieve network slice"}
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
		slices: []configmodels.Slice{networkSlice("slice1")},
	}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
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
	expected := networkSlice("slice1")
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", "YOUR COMMAND"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
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

	slice := networkSlice("slice1")
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

	slice := configmodels.Slice{SliceName: "slice1"}
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
		networkSlice("slice1"),
		networkSlice("slice2"),
		networkSlice("slice_no_gnodeb"),
		networkSlice("slice_no_device_groups"),
	}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}

	syncSubscribersOnSliceCreateOrUpdate = func(slice, prevSlice configmodels.Slice) (int, error) {
		return http.StatusOK, nil
	}

	for _, testSlice := range networkSlices {
		ts := testSlice

		for {
			syncSliceStopMutex.Lock()
			if !SyncSliceStop {
				t.Log("wait wait wait")
				syncSliceStopMutex.Unlock()
				break
			}
			syncSliceStopMutex.Unlock()
		}

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

			if mock.postData[0]["coll"] != sliceDataColl {
				t.Errorf("expected collection %v, got %v", sliceDataColl, mock.postData[0]["coll"])
			}

			expectedFilter := bson.M{"slice-name": ts.SliceName}
			if !reflect.DeepEqual(mock.postData[0]["filter"], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mock.postData[0]["filter"])
			}

			result := mock.postData[0]["data"].(map[string]any)
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

	syncSubscribersOnSliceCreateOrUpdate = func(slice, prevSlice configmodels.Slice) (int, error) {
		return http.StatusOK, nil
	}

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

func TestNetworkSlicePostHandler_NetworkSliceGnbTacValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	syncSubscribersOnSliceCreateOrUpdate = func(slice, prevSlice configmodels.Slice) (int, error) {
		return http.StatusOK, nil
	}

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
			expectedError: "invalid gNodeBs[0].name",
		},
		{
			name:          "Network Slice invalid gNB TAC",
			route:         "/config/v1/network-slice/slice-1",
			inputData:     networkSliceWithGnbParams("slice-1", "valid-gnb", 0),
			expectedCode:  http.StatusBadRequest,
			expectedError: "invalid gNodeBs[0].tac",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tc.inputData)
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
			if !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("expected body to contain error about  `%v`, got `%v`", tc.expectedError, w.Body.String())
			}
		})
	}
}
