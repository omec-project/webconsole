// SPDX-FileCopyrightText: 2023 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockMongoClientNoDeviceGroups struct {
	dbadapter.DBInterface
}

type MockMongoClientOneDeviceGroups struct {
	dbadapter.DBInterface
}

type MockMongoClientManyDeviceGroups struct {
	dbadapter.DBInterface
}

type MockMongoClientNotFoundDeviceGroup struct {
	dbadapter.DBInterface
}

type MockMongoClientFoundDeviceGroup struct {
	dbadapter.DBInterface
}

type MockMongoClientNoNetworkSlice struct {
	dbadapter.DBInterface
}

type MockMongoClientNotFoundNetworkSlice struct {
	dbadapter.DBInterface
}

type MockMongoClientFoundNetworkSlice struct {
	dbadapter.DBInterface
}

type MockMongoClientOneNetworkSlice struct {
	dbadapter.DBInterface
}

type MockMongoClientManyNetworkSlices struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientManyNetworkSlices) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	// The filter likely uses "slice-name" instead of "name"
	if sliceName, ok := filter["slice-name"].(string); ok {
		ns := configmodels.ToBsonM(networkSlice(sliceName))
		if ns == nil {
			return nil, nil
		}
		return ns, nil
	}
	return nil, nil
}

func (m *MockMongoClientManyNetworkSlices) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	names := []string{"slice1", "slice2", "slice3"}
	for _, name := range names {
		ns := configmodels.ToBsonM(networkSlice(name))
		if ns == nil {
			panic("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, nil
}

func (m *MockMongoClientManyNetworkSlices) RestfulAPIPost(coll string, filter bson.M, data map[string]interface{}) (bool, error) {
	return true, nil
}

func (m *MockMongoClientManyNetworkSlices) RestfulAPIDeleteOne(coll string, filter bson.M) error {
	return nil
}

func (m *MockMongoClientManyNetworkSlices) Client() *mongo.Client {
	return nil
}

func (m *MockMongoClientNoDeviceGroups) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (m *MockMongoClientOneDeviceGroups) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	dg := configmodels.ToBsonM(deviceGroup("group1"))
	if dg == nil {
		panic("failed to convert device group to BsonM")
	}
	results = append(results, dg)
	return results, nil
}

func (m *MockMongoClientManyDeviceGroups) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	names := []string{"group1", "group2", "group3"}
	for _, name := range names {
		dg := configmodels.ToBsonM(deviceGroup(name))
		if dg == nil {
			panic("failed to convert device group to BsonM")
		}
		results = append(results, dg)
	}
	return results, nil
}

func (m *MockMongoClientNotFoundDeviceGroup) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockMongoClientFoundDeviceGroup) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	dg := configmodels.ToBsonM(deviceGroup("group1"))
	if dg == nil {
		panic("failed to convert device group to BsonM")
	}
	return dg, nil
}

func (m *MockMongoClientNoNetworkSlice) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (m *MockMongoClientOneNetworkSlice) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ns := configmodels.ToBsonM(networkSlice("slice1"))
	if ns == nil {
		panic("failed to convert network slice to BsonM")
	}
	results = append(results, ns)
	return results, nil
}

func (m *MockMongoClientNotFoundNetworkSlice) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockMongoClientFoundNetworkSlice) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	ns := configmodels.ToBsonM(networkSlice("slice1"))
	if ns == nil {
		panic("failed to convert network slice to BsonM")
	}
	return ns, nil
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

func TestGetDeviceGroupsNoGroups(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	dbadapter.CommonDBClient = &MockMongoClientNoDeviceGroups{}
	GetDeviceGroups(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	if body != "[]" {
		t.Errorf("Expected empty JSON list, got %v", body)
	}
}

func TestGetDeviceGroupsOneGroup(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientOneDeviceGroups{}
	GetDeviceGroups(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `["group1"]`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}

func TestGetDeviceGroupsManyGroup(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientManyDeviceGroups{}
	GetDeviceGroups(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `["group1","group2","group3"]`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}

func TestGetDeviceGroupByNameDoesNotExist(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientNotFoundDeviceGroup{}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != 404 {
		t.Errorf("Expected StatusCode %d, got %d", 404, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	if body != "null" {
		t.Errorf("Expected %v, got %v", "null", body)
	}
}

func TestGetDeviceGroupByNameDoesExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientFoundDeviceGroup{}
	c.Params = append(c.Params, gin.Param{Key: "device-name", Value: "group1"})
	GetDeviceGroupByName(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `{"group-name":"group1","imsis":["1234","5678"],"site-info":"demo","ip-domain-name":"pool1","ip-domain-expanded":{"dnn":"internet","ue-ip-pool":"172.250.1.0/16","dns-primary":"1.1.1.1","dns-secondary":"8.8.8.8","mtu":1460,"ue-dnn-qos":{"dnn-mbr-uplink":10000000,"dnn-mbr-downlink":10000000,"bitrate-unit":"kbps","traffic-class":{"name":"platinum","qci":8,"arp":6,"pdb":300,"pelr":6}}}}`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}

func TestDeviceGroupDeleteHandler_DeviceGroupExistsInNetworkSlices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)
	mock := &MockMongoClientManyNetworkSlices{}

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
	}{
		{
			name:         "Delete DG associated with NSs expects config messages sent for NSs and DG",
			route:        "/config/v1/device-group/group1",
			dbAdapter:    mock,
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDbAdapter := dbadapter.CommonDBClient
			dbadapter.CommonDBClient = tc.dbAdapter
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 10)
			defer func() {
				configChannel = origChannel
				dbadapter.CommonDBClient = originalDbAdapter
			}()
			req, err := http.NewRequest(http.MethodDelete, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			timeout := time.After(2 * time.Second)
			expectedGroupName := "group1"
			expectedSliceNames := []string{"slice1", "slice2", "slice3"}
			for _, expectedSliceName := range expectedSliceNames {
				select {
				case msg := <-configChannel:
					verifyNetworkSliceMessage(t, msg, expectedSliceName, expectedGroupName)
				case <-timeout:
					t.Fatalf("Timeout waiting for network slice message for %s", expectedSliceName)
				}
			}

			select {
			case msg := <-configChannel:
				verifyDeviceGroupMessage(t, msg, expectedGroupName)
			case <-timeout:
				t.Fatal("Timeout waiting for device group deletion message")
			}

			select {
			case msg := <-configChannel:
				t.Errorf("Unexpected extra message in channel: %+v", msg)
			case <-time.After(100 * time.Millisecond):
				// OK - no more messages
			}
		})
	}
}

func verifyNetworkSliceMessage(t *testing.T, msg *configmodels.ConfigMessage, expectedSliceName, expectedGroupName string) {
	t.Helper()
	if msg.MsgType != configmodels.Network_slice {
		t.Errorf("Expected message type %v, got %v", configmodels.Network_slice, msg.MsgType)
	}
	if msg.MsgMethod != configmodels.Post_op {
		t.Errorf("Expected message method %v, got %v", configmodels.Post_op, msg.MsgMethod)
	}
	if msg.SliceName != expectedSliceName {
		t.Errorf("Expected slice name %v, got %v", expectedSliceName, msg.SliceName)
	}
	for _, group := range msg.Slice.SiteDeviceGroup {
		if group == expectedGroupName {
			t.Errorf("Expected %v to be removed from SiteDeviceGroup in slice %s, but it was found",
				expectedGroupName, msg.SliceName)
		}
	}
}

func verifyDeviceGroupMessage(t *testing.T, msg *configmodels.ConfigMessage, expectedGroupName string) {
	t.Helper()
	if msg.MsgType != configmodels.Device_group {
		t.Errorf("Expected message type %v, got %v", configmodels.Device_group, msg.MsgType)
	}
	if msg.MsgMethod != configmodels.Delete_op {
		t.Errorf("Expected message method %v, got %v", configmodels.Delete_op, msg.MsgMethod)
	}
	if msg.DevGroupName != expectedGroupName {
		t.Errorf("Expected device group name %v, got %v", expectedGroupName, msg.DevGroupName)
	}
}

func TestDeviceGroupDeleteHandler_DeviceGroupDoesNotExistInNetworkSlices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
	}{
		{
			name:         "Delete DG not associated with any NS expects only one config message",
			route:        "/config/v1/device-group/group1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDbAdapter := dbadapter.CommonDBClient
			dbadapter.CommonDBClient = tc.dbAdapter
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 10)
			defer func() {
				configChannel = origChannel
				dbadapter.CommonDBClient = originalDbAdapter
			}()
			req, err := http.NewRequest(http.MethodDelete, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			expectedGroupName := "group1"
			timeout := time.After(2 * time.Second)
			select {
			case msg := <-configChannel:
				verifyDeviceGroupMessage(t, msg, expectedGroupName)
			case <-timeout:
				t.Fatal("Timeout waiting for device group deletion message")
			}

			select {
			case msg := <-configChannel:
				t.Errorf("Unexpected extra message in channel: %+v", msg)
			case <-time.After(100 * time.Millisecond):
				// OK - no more messages
			}
		})
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

func TestGetNetworkSlicesNoSlices(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientNoNetworkSlice{}
	GetNetworkSlices(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	if body != "[]" {
		t.Errorf("Expected empty JSON list, got %v", body)
	}
}

func TestGetNetworkSlicesOneSlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientOneNetworkSlice{}
	GetNetworkSlices(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `["slice1"]`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}

func TestGetNetworkSlicesManySlices(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientManyNetworkSlices{}
	GetNetworkSlices(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `["slice1","slice2","slice3"]`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}

func TestGetNetworkSliceByNameDoesNotExist(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientNotFoundNetworkSlice{}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != 404 {
		t.Errorf("Expected StatusCode %d, got %d", 404, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	if body != "null" {
		t.Errorf("Expected %v, got %v", "null", body)
	}
}

func TestGetNetworkSliceByNameDoesExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbadapter.CommonDBClient = &MockMongoClientFoundNetworkSlice{}
	c.Params = append(c.Params, gin.Param{Key: "slice-name", Value: "slice1"})
	GetNetworkSliceByName(c)
	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Expected StatusCode %d, got %d", 200, resp.StatusCode)
	}
	body_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(body_bytes)
	expected := `{"slice-name":"slice1","slice-id":{"sst":"1","sd":"010203"},"site-device-group":["group1","group2"],"site-info":{"site-name":"demo","plmn":{"mcc":"208","mnc":"93"},"gNodeBs":[{"name":"demo-gnb1","tac":1}],"upf":{"upf-name":"upf","upf-port":"8805"}}}`
	if body != expected {
		t.Errorf("Expected %v, got %v", expected, body)
	}
}
