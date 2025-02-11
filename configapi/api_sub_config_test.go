// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type MockMongoClientOneSubscriber struct {
	dbadapter.DBInterface
}

type MockMongoClientManySubscribers struct {
	dbadapter.DBInterface
}

type MockMongoClientDeviceGroupsWithSubscriber struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientOneSubscriber) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber["ueId"] = "208930100007487"
	subscriber["servingPlmnId"] = "12345"
	var subscriberBson bson.M
	tmp, _ := json.Marshal(subscriber)
	json.Unmarshal(tmp, &subscriberBson)

	results = append(results, subscriberBson)
	return results, nil
}

func (m *MockMongoClientManySubscribers) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ueIds := []string{"208930100007487", "208930100007488"}
	plmnIDs := []string{"12345", "54321"}
	for i, ueId := range ueIds {
		subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
		subscriber["ueId"] = ueId
		subscriber["servingPlmnId"] = plmnIDs[i]
		var subscriberBson bson.M
		tmp, _ := json.Marshal(subscriber)
		json.Unmarshal(tmp, &subscriberBson)

		results = append(results, subscriberBson)
	}
	return results, nil
}

func (m *MockMongoClientDeviceGroupsWithSubscriber) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	dg := deviceGroupWithImsis("group1", []string{"208930100007487", "208930100007488"})
	var dgbson bson.M
	tmp, _ := json.Marshal(dg)
	json.Unmarshal(tmp, &dgbson)

	results = append(results, dgbson)
	return results, nil
}

func TestSubscriberGetHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "SubscriberEmptyDB",
			route:        "/api/subscriber",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "[]",
		},
		{
			name:         "Get subscribers list with one element",
			route:        "/api/subscriber",
			dbAdapter:    &MockMongoClientOneSubscriber{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"plmnID":"12345","ueId":"208930100007487"}]`,
		},
		{
			name:         "ManySubscribers",
			route:        "/api/subscriber",
			dbAdapter:    &MockMongoClientManySubscribers{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"plmnID":"12345","ueId":"208930100007487"},{"plmnID":"54321","ueId":"208930100007488"}]`,
		},
		{
			name:         "SubscriberDBError",
			route:        "/api/subscriber",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to retrieve subscribers list"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodGet, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestSubscriberPostHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)

	testCases := []struct {
		name            string
		route           string
		inputData       string
		expectedMessage configmodels.ConfigMessage
	}{
		{
			name:      "Create a new subscriber success",
			route:     "/api/subscriber/imsi-208930100007487",
			inputData: `{"plmnID":"12345", "opc":"8e27b6af0e692e750f32667a3b14605d","key":"8baf473f2f8fd09487cccbd7097c6862", "sequenceNumber":"16f3b3f70fc2"}`,
			expectedMessage: configmodels.ConfigMessage{
				MsgType:   configmodels.Sub_data,
				MsgMethod: configmodels.Post_op,
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
				Imsi: "imsi-208930100007487",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 1)
			defer func() { configChannel = origChannel }()
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			expectedCode := http.StatusCreated
			expectedBody := "{}"

			if expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != expectedBody {
				t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
			}
			select {
			case msg := <-configChannel:

				if msg.MsgType != tc.expectedMessage.MsgType {
					t.Errorf("expected MsgType %+v, but got %+v", tc.expectedMessage.MsgType, msg.MsgType)
				}
				if msg.MsgMethod != tc.expectedMessage.MsgMethod {
					t.Errorf("expected MsgMethod %+v, but got %+v", tc.expectedMessage.MsgMethod, msg.MsgMethod)
				}
				if !reflect.DeepEqual(tc.expectedMessage.AuthSubData, msg.AuthSubData) {
					t.Errorf("expected AuthSubData %+v, but got %+v", tc.expectedMessage.AuthSubData, msg.AuthSubData)
				}
				if tc.expectedMessage.Imsi != msg.Imsi {
					t.Errorf("expected IMSI %+v, but got %+v", tc.expectedMessage.Imsi, msg.Imsi)
				}
			default:
				t.Error("expected message in configChannel, but none received")
			}
		})
	}
}

func TestSubscriberDeleteNoDeviceGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	dbAdapter := &MockMongoClientEmptyDB{}
	dbadapter.CommonDBClient = dbAdapter
	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusNoContent
	expectedBody := ""
	expectedMessage := configmodels.ConfigMessage{
		MsgType:   configmodels.Sub_data,
		MsgMethod: configmodels.Delete_op,
		Imsi:      "imsi-208930100007487",
	}
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 2)
	defer func() { configChannel = origChannel }()
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if expectedBody != w.Body.String() {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}
	select {
		case msg := <-configChannel:
			if expectedMessage.MsgType != msg.MsgType {
				t.Errorf("expected MsgType %+v, but got %+v", expectedMessage.MsgType, msg.MsgType)
			}
			if expectedMessage.MsgMethod != msg.MsgMethod {
				t.Errorf("expected MsgMethod %+v, but got %+v", expectedMessage.MsgMethod, msg.MsgMethod)
			}
			if expectedMessage.Imsi != msg.Imsi {
				t.Errorf("expected IMSI %+v, but got %+v", expectedMessage.Imsi, msg.Imsi)
			}
		default:
			t.Error("expected message in configChannel, but none received")
	}
}

func TestSubscriberDeleteWithDeviceGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	dbAdapter := &MockMongoClientDeviceGroupsWithSubscriber{}
	dbadapter.CommonDBClient = dbAdapter
	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusNoContent
	expectedBody := ""
	expectedMessage := configmodels.ConfigMessage{
		MsgType:   configmodels.Device_group,
		MsgMethod: configmodels.Post_op,
		DevGroupName: "group1",
		DevGroup: deviceGroupWithoutImsi(),
	}
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 2)
	defer func() { configChannel = origChannel }()
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if expectedBody != w.Body.String() {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}
	select {
		case msg := <-configChannel:
			if expectedMessage.MsgType != msg.MsgType {
				t.Errorf("expected MsgType %+v, but got %+v", expectedMessage.MsgType, msg.MsgType)
			}
			if expectedMessage.MsgMethod != msg.MsgMethod {
				t.Errorf("expected MsgMethod %+v, but got %+v", expectedMessage.MsgMethod, msg.MsgMethod)
			}
			if expectedMessage.DevGroupName != msg.DevGroupName {
				t.Errorf("expected device group name %+v, but got %+v", expectedMessage.DevGroupName, msg.DevGroupName)
			}
			if !reflect.DeepEqual(expectedMessage.DevGroup.Imsis, msg.DevGroup.Imsis) {
				t.Errorf("expected IMSIs in device group: %+v, but got %+v", expectedMessage.DevGroup.Imsis, msg.DevGroup.Imsis)
			}
		default:
			t.Error("expected message in configChannel, but none received")
	}
}

func deviceGroupWithImsis(name string, imsis []string) configmodels.DeviceGroups {
	trafficClass := configmodels.TrafficClassInfo{
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
		TrafficClass:   &trafficClass,
	}
	ipDomain := configmodels.DeviceGroupsIpDomainExpanded{
		Dnn:          "internet",
		UeIpPool:     "172.250.1.0/16",
		DnsPrimary:   "1.1.1.1",
		DnsSecondary: "8.8.8.8",
		Mtu:          1460,
		UeDnnQos:     &qos,
	}
	deviceGroup := configmodels.DeviceGroups{
		DeviceGroupName:  name,
		Imsis:            imsis,
		SiteInfo:         "demo",
		IpDomainName:     "pool1",
		IpDomainExpanded: ipDomain,
	}
	return deviceGroup
}

func deviceGroupWithoutImsi() *configmodels.DeviceGroups {
	tmp := deviceGroupWithImsis("group1", []string{"208930100007487", "208930100007488"})
	tmp.Imsis = slices.Delete(tmp.Imsis, 0, 1)
	return &tmp
}
