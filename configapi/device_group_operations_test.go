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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	// Other collections (e.g. sliceDataColl, looked up when syncing a device group's associated
	// slice) must not be answered with device group documents, which cannot unmarshal as anything
	// else and would otherwise mask that path failing to see "no data" as it should.
	if coll != devGroupDataColl || len(db.configuredDeviceGroups) == 0 {
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
	if coll != devGroupDataColl {
		return nil, nil
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
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	}
	db.postData = append(db.postData, params)
	return true, nil
}

func (db *DeviceGroupMockDBClient) RestfulAPIDeleteOne(coll string, filter bson.M) error {
	params := map[string]any{
		collKey:   coll,
		filterKey: filter,
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
		BitrateUnit:    bitrateUnitKbps,
		TrafficClass:   &traffic_class,
	}
	ipdomain := configmodels.DeviceGroupsIpDomainExpanded{
		Dnn:          dnnInternet,
		UeIpPool:     "172.250.1.0/16",
		DnsPrimary:   "1.1.1.1",
		DnsSecondary: "8.8.8.8",
		Mtu:          1460,
		UeDnnQos:     &qos,
	}
	deviceGroup := configmodels.DeviceGroups{
		DeviceGroupName: name,
		Imsis:           []string{"1234", "5678"},
		SiteInfo:        demoSiteName,
		IpDomainName:    "pool1",
		IpDomainsExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
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
				deviceGroup(testGroupName),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{testGroupName},
		},
		{
			name: "Many device groups returns a list with many names",
			configuredDeviceGroups: []configmodels.DeviceGroups{
				deviceGroup(testGroupName),
				deviceGroup("group2"),
				deviceGroup("group3"),
			},
			expectedCode:   http.StatusOK,
			expectedResult: []string{testGroupName, "group2", "group3"},
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
	c.Params = append(c.Params, gin.Param{Key: groupNameKey, Value: testGroupName})
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
	c.Params = append(c.Params, gin.Param{Key: groupNameKey, Value: testGroupName})
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

	expected := map[string]string{errorKey: errMsgRetrieveDeviceGroup}
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
		configuredDeviceGroups: []configmodels.DeviceGroups{deviceGroup(testGroupName)},
	}
	c.Params = append(c.Params, gin.Param{Key: groupNameKey, Value: testGroupName})
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
	expected := deviceGroup(testGroupName)
	// The stored rates are bps whatever unit the fixture carries beside them, and the GET now says
	// so -- see TestGetDeviceGroupByNameLabelsStoredRatesAsBps for why.
	expected.IpDomainsExpanded[0].UeDnnQos.BitrateUnit = bitrateUnitBps
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func Test_handleDeviceGroupPost(t *testing.T) {
	deviceGroups := []configmodels.DeviceGroups{
		deviceGroup(testGroupName),
		deviceGroup("group2"),
		deviceGroup("group_no_imsis"),
		deviceGroup("group_no_traf_class"),
		deviceGroup("group_no_qos"),
	}
	deviceGroups[2].Imsis = []string{}
	if len(deviceGroups[3].IpDomainsExpanded) > 0 {
		deviceGroups[3].IpDomainsExpanded[0].UeDnnQos.TrafficClass = nil
	}
	if len(deviceGroups[4].IpDomainsExpanded) > 0 {
		deviceGroups[4].IpDomainsExpanded[0].UeDnnQos = nil
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

			if mockDB.postData[0][collKey] != devGroupDataColl {
				t.Errorf("expected collection %v, got %v", devGroupDataColl, mockDB.postData[0][collKey])
			}

			expectedFilter := bson.M{groupNameKey: dg.DeviceGroupName}
			if !reflect.DeepEqual(mockDB.postData[0][filterKey], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mockDB.postData[0][filterKey])
			}

			result := mockDB.postData[0][dataKey].(map[string]any)
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
		deviceGroup(testGroupName),
		deviceGroup("group2"),
		deviceGroup("group_no_imsis"),
		deviceGroup("group_no_traf_class"),
		deviceGroup("group_no_qos"),
	}
	deviceGroups[2].Imsis = []string{}
	if len(deviceGroups[3].IpDomainsExpanded) > 0 {
		deviceGroups[3].IpDomainsExpanded[0].UeDnnQos.TrafficClass = nil
	}
	if len(deviceGroups[4].IpDomainsExpanded) > 0 {
		deviceGroups[4].IpDomainsExpanded[0].UeDnnQos = nil
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

			if mock.postData[0][collKey] != devGroupDataColl {
				t.Errorf("expected collection %v, got %v", devGroupDataColl, mock.postData[0][collKey])
			}

			expectedFilter := bson.M{groupNameKey: dg.DeviceGroupName}
			if !reflect.DeepEqual(mock.postData[0][filterKey], expectedFilter) {
				t.Errorf("expected filter %v, got %v", expectedFilter, mock.postData[0][filterKey])
			}

			result := mock.postData[0][dataKey].(map[string]any)
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

	err := handleDeviceGroupDelete(testGroupName)
	if err != nil {
		t.Fatalf("handleDeviceGroupDelete failed: %v", err)
	}

	if len(dbClientMock.deleteData) == 0 {
		t.Fatal("no delete operation was recorded")
	}

	expectedColl := devGroupDataColl
	if dbClientMock.deleteData[0][collKey] != expectedColl {
		t.Errorf("expected collection %v, got %v", expectedColl, dbClientMock.deleteData[0][collKey])
	}

	expectedFilter := bson.M{groupNameKey: testGroupName}
	if !reflect.DeepEqual(dbClientMock.deleteData[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, dbClientMock.deleteData[0][filterKey])
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

// deviceGroupWithRates is a group whose single IP domain carries the given rates and unit, so a
// test can drive the ingest path with the values it is about rather than the fixture's.
func deviceGroupWithRates(name, unit string, uplink, downlink int64) configmodels.DeviceGroups {
	dg := deviceGroup(name)
	dg.IpDomainsExpanded[0].UeDnnQos.BitrateUnit = unit
	dg.IpDomainsExpanded[0].UeDnnQos.DnnMbrUplink = uplink
	dg.IpDomainsExpanded[0].UeDnnQos.DnnMbrDownlink = downlink
	return dg
}

func postDeviceGroup(t *testing.T, dg configmodels.DeviceGroups) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	jsonBody, err := json.Marshal(dg)
	if err != nil {
		t.Fatalf("failed to marshal device group: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		"/config/v1/device-group/"+dg.DeviceGroupName, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// DeviceGroupSliceAwareMockDBClient answers slice lookups by collection, unlike
// DeviceGroupMockDBClient which always returns the same stored device group regardless of
// collection.
type DeviceGroupSliceAwareMockDBClient struct {
	dbadapter.DBInterface
	slices   []configmodels.Slice
	postData []map[string]any
}

func (db *DeviceGroupSliceAwareMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	return nil, nil
}

func (db *DeviceGroupSliceAwareMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if coll != sliceDataColl {
		return nil, nil
	}
	var results []map[string]any
	for _, s := range db.slices {
		results = append(results, configmodels.ToBsonM(s))
	}
	return results, nil
}

func (db *DeviceGroupSliceAwareMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.postData = append(db.postData, map[string]any{collKey: collName, filterKey: filter, dataKey: postData})
	return true, nil
}

// The first digits of an IMSI are its home PLMN, so a device group already attached to a slice
// must not accept a subscriber whose IMSI belongs to a different PLMN.
func TestDeviceGroupPostHandler_RejectsImsiNotMatchingAssociatedSlicePlmn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	associatedSlice := networkSlice(testSliceName)
	associatedSlice.SiteDeviceGroup = []string{testGroupName}

	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	mock := &DeviceGroupSliceAwareMockDBClient{slices: []configmodels.Slice{associatedSlice}}
	dbadapter.CommonDBClient = mock

	newDeviceGroup := deviceGroup(testGroupName)
	newDeviceGroup.Imsis = []string{"999990000000001"}
	jsonBody, err := json.Marshal(newDeviceGroup)
	if err != nil {
		t.Fatalf("failed to marshal device group: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/device-group/"+testGroupName, bytes.NewReader(jsonBody))
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

// A rate that cannot be served as configured is refused, where it used to be accepted and stored as
// something else entirely: a negative rate became math.MaxInt64, the largest rate the element can
// carry, and a Gbps rate large enough to wrap the field was clamped to the same value by the sign
// of its own wrapped product.
func TestDeviceGroupPostRefusesRatesThatCannotBeServed(t *testing.T) {
	testCases := []struct {
		name         string
		unit         string
		uplink       int64
		downlink     int64
		expectedCode int
	}{
		{"an ordinary rate", bitrateUnitMbps, 100, 200, http.StatusOK},
		{"the largest rate that can be served", bitrateUnitGbps, 65535, 65535, http.StatusOK},
		{"one unit past it", bitrateUnitGbps, 65536, 100, http.StatusBadRequest},
		{"negative uplink", bitrateUnitMbps, -1, 100, http.StatusBadRequest},
		{"negative downlink", bitrateUnitMbps, 100, -1, http.StatusBadRequest},
		{"a product that wraps the field", bitrateUnitGbps, 10000000000, 100, http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbMock := &DeviceGroupMockDBClient{}
			dbadapter.CommonDBClient = dbMock

			w := postDeviceGroup(t, deviceGroupWithRates(testGroupName, tc.unit, tc.uplink, tc.downlink))

			if w.Code != tc.expectedCode {
				t.Errorf("expected %d, got %d: %s", tc.expectedCode, w.Code, w.Body.String())
			}
			// A refusal that still wrote the group would leave the rate it refused in the database.
			if tc.expectedCode != http.StatusOK && len(dbMock.postData) != 0 {
				t.Errorf("the device group was stored although the request was refused: %+v", dbMock.postData)
			}
		})
	}
}

// The stored rates are bps, so the stored unit has to be bps as well. Left as posted it makes the
// document multiply its own rates every time an operator's tool reads it and writes it back.
func TestDeviceGroupPostStoresRatesInBpsAndSaysSo(t *testing.T) {
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	dbMock := &DeviceGroupMockDBClient{}
	dbadapter.CommonDBClient = dbMock

	w := postDeviceGroup(t, deviceGroupWithRates(testGroupName, "kbps", 2000000, 1000000))
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if len(dbMock.postData) == 0 {
		t.Fatal("expected the device group to be stored")
	}
	var stored configmodels.DeviceGroups
	if err := json.Unmarshal(configmodels.MapToByte(dbMock.postData[0][dataKey].(map[string]any)), &stored); err != nil {
		t.Fatalf("failed to unmarshal the stored device group: %v", err)
	}
	qos := stored.IpDomainsExpanded[0].UeDnnQos
	if qos.DnnMbrUplink != 2000000000 || qos.DnnMbrDownlink != 1000000000 {
		t.Errorf("stored rates = %d/%d bps, want 2000000000/1000000000", qos.DnnMbrUplink, qos.DnnMbrDownlink)
	}
	if qos.BitrateUnit != bitrateUnitBps {
		t.Errorf("stored bitrate-unit = %q, want %q: the stored rates are bps", qos.BitrateUnit, bitrateUnitBps)
	}
}

// What a group written before the unit was stored to match looks like: rates already in bps, beside
// the unit the operator posted. A GET has to say bps, and posting that document back has to store
// the same rates rather than multiply them again.
func TestGetDeviceGroupByNameLabelsStoredRatesAsBps(t *testing.T) {
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	// A kbps group, not an Mbps one: 2000000 re-multiplied is 2000000000, which is still a rate the
	// ingest path accepts. The label is what stops it, so the assertion that catches the drift is
	// the comparison of the stored rates -- an Mbps group this size is refused by the rate
	// validation instead, which is a different failure and would hide this one.
	stored := deviceGroupWithRates(testGroupName, "kbps", 2000000, 1000000)
	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		configuredDeviceGroups: []configmodels.DeviceGroups{stored},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: groupNameKey, Value: testGroupName})
	GetDeviceGroupByName(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
	var returned configmodels.DeviceGroups
	if err := json.Unmarshal(w.Body.Bytes(), &returned); err != nil {
		t.Fatalf("failed to unmarshal the returned device group: %v", err)
	}
	qos := returned.IpDomainsExpanded[0].UeDnnQos
	if qos.BitrateUnit != bitrateUnitBps {
		t.Errorf("bitrate-unit = %q, want %q: the stored rates are bps", qos.BitrateUnit, bitrateUnitBps)
	}
	if qos.DnnMbrUplink != 2000000 || qos.DnnMbrDownlink != 1000000 {
		t.Errorf("the returned rates were altered: %d/%d", qos.DnnMbrUplink, qos.DnnMbrDownlink)
	}

	postMock := &DeviceGroupMockDBClient{}
	dbadapter.CommonDBClient = postMock
	if code := postDeviceGroup(t, returned).Code; code != http.StatusOK {
		t.Fatalf("posting back what the GET returned was refused with %d", code)
	}
	if len(postMock.postData) == 0 {
		t.Fatal("expected the device group to be stored")
	}
	var storedAgain configmodels.DeviceGroups
	if err := json.Unmarshal(configmodels.MapToByte(postMock.postData[0][dataKey].(map[string]any)), &storedAgain); err != nil {
		t.Fatalf("failed to unmarshal the stored device group: %v", err)
	}
	if !reflect.DeepEqual(storedAgain.IpDomainsExpanded, returned.IpDomainsExpanded) {
		t.Errorf("posting back what a GET returned changed the rates: %+v was stored as %+v",
			returned.IpDomainsExpanded, storedAgain.IpDomainsExpanded)
	}
}

// The relabelling is on every read of a stored group, not only the one the API serves.
// getDeviceGroupByName is what the slice path loads a group with, and aggregateQoS compares the
// units of the groups it is given: a stale one there makes it report an inconsistency between two
// groups whose rates are both already bps, and pick that stale unit for the aggregate.
func TestGetDeviceGroupByNameHelperLabelsStoredRatesAsBps(t *testing.T) {
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()

	stored := deviceGroupWithRates(testGroupName, "kbps", 2000000, 1000000)
	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		configuredDeviceGroups: []configmodels.DeviceGroups{stored},
	}

	loaded, err := getDeviceGroupByName(testGroupName)
	if err != nil {
		t.Fatalf("failed to look up device group: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected the device group to be loaded")
	}
	qos := loaded.IpDomainsExpanded[0].UeDnnQos
	if qos.BitrateUnit != bitrateUnitBps {
		t.Errorf("bitrate-unit = %q, want %q: the stored rates are bps", qos.BitrateUnit, bitrateUnitBps)
	}
	if qos.DnnMbrUplink != 2000000 || qos.DnnMbrDownlink != 1000000 {
		t.Errorf("the loaded rates were altered: %d/%d", qos.DnnMbrUplink, qos.DnnMbrDownlink)
	}

	// The property the label exists for on this path: two groups stored under different units are
	// both bps, so aggregating them reports one unit rather than a disagreement.
	other := deviceGroupWithRates("group2", bitrateUnitMbps, 3000000, 1000000)
	dbadapter.CommonDBClient = &DeviceGroupMockDBClient{
		configuredDeviceGroups: []configmodels.DeviceGroups{other},
	}
	otherLoaded, err := getDeviceGroupByName("group2")
	if err != nil {
		t.Fatalf("failed to look up device group: %v", err)
	}
	if otherLoaded == nil {
		t.Fatal("expected the second device group to be loaded")
	}
	aggregated := aggregateQoS([]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
		*qos, *otherLoaded.IpDomainsExpanded[0].UeDnnQos,
	})
	if aggregated.BitrateUnit != bitrateUnitBps {
		t.Errorf("aggregated bitrate-unit = %q, want %q", aggregated.BitrateUnit, bitrateUnitBps)
	}
}
