// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	PostDataCommon *[]map[string]interface{}
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
	results = append(results, subscriber)
	return results, nil
}

func (m *MockMongoClientOneSubscriber) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	if m.PostDataCommon != nil {
		*m.PostDataCommon = append(*m.PostDataCommon, map[string]interface{}{
			"coll":   collName,
			"filter": filter,
		})
	}

	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber["ueId"] = "208930100007487"
	subscriber["servingPlmnId"] = "12345"
	return subscriber, nil
}

func (m *MockMongoClientDeviceGroupsWithSubscriber) RestfulAPIPost(coll string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return true, nil
}

func (m *MockMongoClientDeviceGroupsWithSubscriber) RestfulAPIDeleteOne(coll string, filter bson.M) error {
	return nil
}

func (m *MockMongoClientManySubscribers) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ueIds := []string{"208930100007487", "208930100007488"}
	plmnIDs := []string{"12345", "54321"}
	for i, ueId := range ueIds {
		subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
		subscriber["ueId"] = ueId
		subscriber["servingPlmnId"] = plmnIDs[i]
		results = append(results, subscriber)
	}
	return results, nil
}

func (m *MockMongoClientDeviceGroupsWithSubscriber) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	dg := configmodels.ToBsonM(deviceGroupWithImsis("group1", []string{"208930100007487", "208930100007488"}))
	if dg == nil {
		panic("failed to convert device group to BsonM")
	}
	results = append(results, dg)
	return results, nil
}

func (m *MockMongoClientDeviceGroupsWithSubscriber) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	if coll == "device_group" && filter["deviceGroupName"] == "group1" {
		return map[string]interface{}{
			"deviceGroupName": "group1",
			"imsi":            []string{"imsi-208930100007487"},
		}, nil
	}
	return nil, nil
}

type MockAuthDBClientEmpty struct {
	dbadapter.DBInterface
	PostDataAuth *[]map[string]interface{}
}

func (m *MockAuthDBClientEmpty) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	if m.PostDataAuth != nil {
		*m.PostDataAuth = append(*m.PostDataAuth, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}
	return nil, nil
}

type MockAuthDBClientWithData struct {
	dbadapter.DBInterface
	PostDataAuth *[]map[string]interface{}
}

func (m *MockAuthDBClientWithData) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	if m.PostDataAuth != nil {
		*m.PostDataAuth = append(*m.PostDataAuth, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}
	authSubscription := configmodels.ToBsonM(models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "c9e8763286b5b9ffbdf56e1297d0887b",
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            "981d464c7c52eb6e5036234984ad0bcf",
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "5122250214c33e723a5dd523fc145fc0",
		},
		SequenceNumber: "16f3b3f70fc2",
	})
	if authSubscription == nil {
		panic("failed to convert subscriber to BsonM")
	}
	return authSubscription, nil
}

type MockCommonDBClientEmpty struct {
	dbadapter.DBInterface
	PostDataCommon *[]map[string]interface{}
}

func (m *MockCommonDBClientEmpty) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	if m.PostDataCommon != nil {
		*m.PostDataCommon = append(*m.PostDataCommon, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}
	return nil, nil
}

func (m *MockCommonDBClientEmpty) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	if m.PostDataCommon != nil {
		*m.PostDataCommon = append(*m.PostDataCommon, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}
	return nil, nil
}

type MockCommonDBClientWithData struct {
	dbadapter.DBInterface
	PostDataCommon *[]map[string]interface{}
}

func (m *MockCommonDBClientWithData) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	if m.PostDataCommon != nil {
		*m.PostDataCommon = append(*m.PostDataCommon, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}

	switch coll {
	case "subscriptionData.provisionedData.amData":
		amDataData := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{
			Gpsis: []string{
				"msisdn-0900000000",
			},
			Nssai: &models.Nssai{
				DefaultSingleNssais: []models.Snssai{
					{
						Sd:  "010203",
						Sst: 1,
					},
				},
				SingleNssais: []models.Snssai{
					{
						Sd:  "010203",
						Sst: 1,
					},
				},
			},
			SubscribedUeAmbr: &models.AmbrRm{
				Downlink: "1000 Kbps",
				Uplink:   "1000 Kbps",
			},
		})
		if amDataData == nil {
			panic("failed to convert amDataData to BsonM")
		}
		return amDataData, nil

	case "policyData.ues.amData":
		amPolicyData := configmodels.ToBsonM(models.AmPolicyData{
			SubscCats: []string{
				"aether",
			},
		})
		if amPolicyData == nil {
			panic("failed to convert amPolicyData to BsonM")
		}
		return amPolicyData, nil

	case "policyData.ues.smData":
		smPolicyData := configmodels.ToBsonM(models.SmPolicyData{
			SmPolicySnssaiData: map[string]models.SmPolicySnssaiData{
				"01010203": {
					Snssai: &models.Snssai{
						Sd:  "010203",
						Sst: 1,
					},
					SmPolicyDnnData: map[string]models.SmPolicyDnnData{
						"internet": {
							Dnn: "internet",
						},
					},
				},
			},
		})
		if smPolicyData == nil {
			panic("failed to convert smPolicyData to BsonM")
		}
		return smPolicyData, nil

	case "subscriptionData.provisionedData.smfSelectionSubscriptionData":
		smfSelData := configmodels.ToBsonM(models.SmfSelectionSubscriptionData{
			SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
				"01010203": {
					DnnInfos: []models.DnnInfo{
						{
							Dnn: "internet",
						},
					},
				},
			},
		})
		if smfSelData == nil {
			panic("failed to convert smfSelData to BsonM")
		}
		return smfSelData, nil

	default:
		return nil, fmt.Errorf("collection %s not found", coll)
	}
}

func (m *MockCommonDBClientWithData) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	if m.PostDataCommon != nil {
		*m.PostDataCommon = append(*m.PostDataCommon, map[string]interface{}{
			"coll":   coll,
			"filter": filter,
		})
	}
	smDataData := []models.SessionManagementSubscriptionData{
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd:  "010203",
			},
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode:  models.SscMode__1,
						AllowedSscModes: []models.SscMode{models.SscMode__1},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "1000 Kbps",
						Uplink:   "1000 Kbps",
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
				},
			},
		},
	}
	result := make([]map[string]interface{}, len(smDataData))
	for i, smData := range smDataData {
		result[i] = map[string]interface{}{
			"SingleNssai":       smData.SingleNssai,
			"DnnConfigurations": smData.DnnConfigurations,
		}
	}
	return result, nil
}

func TestGetSubscriberByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	postDataCommon := make([]map[string]interface{}, 0)
	postDataAuth := make([]map[string]interface{}, 0)

	tests := []struct {
		name                          string
		ueId                          string
		route                         string
		commonDbAdapter               dbadapter.DBInterface
		authDbAdapter                 dbadapter.DBInterface
		expectedHTTPStatus            int
		expectedFullResponse          map[string]interface{}
		expectedCommonPostDataDetails []map[string]interface{}
		expectedAuthPostDataDetails   []map[string]interface{}
	}{
		{
			name:                 "No subscriber data found",
			ueId:                 "imsi-2089300007487",
			route:                "/api/subscriber/:ueId",
			commonDbAdapter:      &MockCommonDBClientEmpty{PostDataCommon: &postDataCommon},
			authDbAdapter:        &MockAuthDBClientEmpty{PostDataAuth: &postDataAuth},
			expectedHTTPStatus:   http.StatusNotFound,
			expectedFullResponse: map[string]interface{}{"error": "subscriber with ID imsi-2089300007487 not found"},
			expectedCommonPostDataDetails: []map[string]interface{}{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smfSelectionSubscriptionData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.amData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.smData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
			},
			expectedAuthPostDataDetails: []map[string]interface{}{
				{"coll": "subscriptionData.authenticationData.authenticationSubscription", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
			},
		},

		{
			name:               "Valid subscriber data retrieved",
			ueId:               "imsi-2089300007487",
			commonDbAdapter:    &MockCommonDBClientWithData{PostDataCommon: &postDataCommon},
			authDbAdapter:      &MockAuthDBClientWithData{PostDataAuth: &postDataAuth},
			route:              "/api/subscriber/:ueId",
			expectedHTTPStatus: http.StatusOK,
			expectedFullResponse: map[string]interface{}{
				"AccessAndMobilitySubscriptionData": map[string]interface{}{
					"gpsis": []interface{}{"msisdn-0900000000"},
					"nssai": map[string]interface{}{
						"defaultSingleNssais": []interface{}{
							map[string]interface{}{"sd": "010203", "sst": 1},
						},
						"singleNssais": []interface{}{
							map[string]interface{}{"sd": "010203", "sst": 1},
						},
					},
					"subscribedUeAmbr": map[string]interface{}{
						"downlink": "1000 Kbps",
						"uplink":   "1000 Kbps",
					},
				},
				"AmPolicyData": map[string]interface{}{
					"subscCats": []interface{}{"aether"},
				},
				"AuthenticationSubscription": map[string]interface{}{
					"authenticationManagementField": "8000",
					"authenticationMethod":          "5G_AKA",
					"milenage": map[string]interface{}{
						"op": map[string]interface{}{
							"encryptionAlgorithm": 0,
							"encryptionKey":       0,
							"opValue":             "c9e8763286b5b9ffbdf56e1297d0887b",
						},
					},
					"opc": map[string]interface{}{
						"encryptionAlgorithm": 0,
						"encryptionKey":       0,
						"opcValue":            "981d464c7c52eb6e5036234984ad0bcf",
					},
					"permanentKey": map[string]interface{}{
						"encryptionAlgorithm": 0,
						"encryptionKey":       0,
						"permanentKeyValue":   "5122250214c33e723a5dd523fc145fc0",
					},
					"sequenceNumber": "16f3b3f70fc2",
				},
				"FlowRules": nil,
				"SessionManagementSubscriptionData": []interface{}{
					map[string]interface{}{
						"dnnConfigurations": map[string]interface{}{
							"internet": map[string]interface{}{
								"5gQosProfile": map[string]interface{}{
									"5qi":           9,
									"arp":           map[string]interface{}{"preemptCap": "", "preemptVuln": "", "priorityLevel": 8},
									"priorityLevel": 8,
								},
								"pduSessionTypes": map[string]interface{}{
									"allowedSessionTypes": []interface{}{"IPV4"},
									"defaultSessionType":  "IPV4",
								},
								"sessionAmbr": map[string]interface{}{
									"downlink": "1000 Kbps",
									"uplink":   "1000 Kbps",
								},
								"sscModes": map[string]interface{}{
									"allowedSscModes": []interface{}{"SSC_MODE_1"},
									"defaultSscMode":  "SSC_MODE_1",
								},
							},
						},
						"singleNssai": map[string]interface{}{
							"sd":  "010203",
							"sst": 1,
						},
					},
				},
				"SmPolicyData": map[string]interface{}{
					"smPolicySnssaiData": map[string]interface{}{
						"01010203": map[string]interface{}{
							"smPolicyDnnData": map[string]interface{}{
								"internet": map[string]interface{}{
									"dnn": "internet",
								},
							},
							"snssai": map[string]interface{}{
								"sd":  "010203",
								"sst": 1,
							},
						},
					},
				},
				"SmfSelectionSubscriptionData": map[string]interface{}{
					"subscribedSnssaiInfos": map[string]interface{}{
						"01010203": map[string]interface{}{
							"dnnInfos": []interface{}{
								map[string]interface{}{
									"dnn": "internet",
								},
							},
						},
					},
				},
				"plmnID": "",
				"ueId":   "imsi-2089300007487",
			},
			expectedCommonPostDataDetails: []map[string]interface{}{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smfSelectionSubscriptionData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.amData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.smData", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
			},
			expectedAuthPostDataDetails: []map[string]interface{}{
				{"coll": "subscriptionData.authenticationData.authenticationSubscription", "filter": map[string]interface{}{"ueId": "imsi-2089300007487"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postDataCommon = nil
			postDataAuth = nil
			originalAuthDBClient := dbadapter.AuthDBClient
			originalCommonDBClient := dbadapter.CommonDBClient
			dbadapter.CommonDBClient = tt.commonDbAdapter
			dbadapter.AuthDBClient = tt.authDbAdapter
			defer func() {
				dbadapter.CommonDBClient = originalCommonDBClient
				dbadapter.AuthDBClient = originalAuthDBClient
			}()

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/subscriber/%s", tt.ueId), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedHTTPStatus {
				t.Errorf("Expected HTTP status %d, got %d", tt.expectedHTTPStatus, w.Code)
			}

			responseContent := w.Body.String()
			var actual map[string]interface{}
			if err := json.Unmarshal([]byte(responseContent), &actual); err != nil {
				t.Fatalf("Failed to unmarshal actual response: %v. Raw response: %s", err, responseContent)
			}
			expectedResponse, err := json.Marshal(tt.expectedFullResponse)
			if err != nil {
				t.Fatalf("failed to marshal expected response: %v", err)
			}
			actualResponse, err := json.Marshal(actual)
			if err != nil {
				t.Fatalf("failed to marshal actual response: %v", err)
			}
			if !reflect.DeepEqual(expectedResponse, actualResponse) {
				t.Errorf("Mismatch in response:\nExpected:\n%s\nGot:\n%s\n", string(expectedResponse), string(actualResponse))
			}

			expectedCommonData, err := json.Marshal(tt.expectedCommonPostDataDetails)
			if err != nil {
				t.Fatalf("failed to marshal expected post data details: %v", err)
			}
			gotCommonData, err := json.Marshal(postDataCommon)
			if err != nil {
				t.Fatalf("failed to marshal actual post data details: %v", err)
			}
			if !reflect.DeepEqual(expectedCommonData, gotCommonData) {
				t.Errorf("Expected CommonPostData `%v`, but got `%v`", tt.expectedCommonPostDataDetails, postDataCommon)
			}

			expectedAuthData, err := json.Marshal(tt.expectedAuthPostDataDetails)
			if err != nil {
				t.Fatalf("failed to marshal expected auth post data details: %v", err)
			}
			gotAuthData, err := json.Marshal(postDataAuth)
			if err != nil {
				t.Fatalf("failed to marshal actual auth post data details: %v", err)
			}
			if !reflect.DeepEqual(expectedAuthData, gotAuthData) {
				t.Errorf("Expected AuthPostData `%v`, but got `%v`", tt.expectedAuthPostDataDetails, postDataAuth)
			}
		})
	}
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
			origDBClient := dbadapter.CommonDBClient
			dbadapter.CommonDBClient = tc.dbAdapter
			defer func() { dbadapter.CommonDBClient = origDBClient }()
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

func TestSubscriberPostHandlersNoExistingSubscriber(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)

	postDataCommon := make([]map[string]interface{}, 0)
	route := "/api/subscriber/imsi-208930100007487"
	inputData := map[string]string{
		"plmnID":         "12345",
		"opc":            "8e27b6af0e692e750f32667a3b14605d",
		"key":            "8baf473f2f8fd09487cccbd7097c6862",
		"sequenceNumber": "16f3b3f70fc2",
	}
	authSubsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "",
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
	jsonData, err := json.Marshal(inputData)
	if err != nil {
		t.Fatalf("failed to marshal input data to JSON: %v", err)
	}
	commonDbAdapter := &MockMongoClientNoSubscriberInDB{PostDataCommon: &postDataCommon}
	authDbAdapter := &MockMongoClientAuthDB{}
	origDBClient := dbadapter.CommonDBClient
	origAuthDBClient := dbadapter.AuthDBClient
	dbadapter.CommonDBClient = commonDbAdapter
	dbadapter.AuthDBClient = authDbAdapter
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 1)
	defer func() {
		configChannel = origChannel
		dbadapter.CommonDBClient = origDBClient
		dbadapter.AuthDBClient = origAuthDBClient
	}()
	expectedCode := http.StatusCreated
	expectedBody := `{}`
	expectedCommonPostDataDetails := []map[string]interface{}{
		{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]interface{}{"ueId": "imsi-208930100007487"}},
	}
	expectedMessage := &configmodels.ConfigMessage{
		MsgType:     configmodels.Sub_data,
		MsgMethod:   configmodels.Post_op,
		AuthSubData: &authSubsData,
		Imsi:        "imsi-208930100007487",
	}

	req, err := http.NewRequest(http.MethodPost, route, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != expectedCode {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if w.Body.String() != expectedBody {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}

	expectedCommonData, err := json.Marshal(expectedCommonPostDataDetails)
	if err != nil {
		t.Fatalf("failed to marshal expected post data details: %v", err)
	}
	gotCommonData, err := json.Marshal(postDataCommon)
	if err != nil {
		t.Fatalf("failed to marshal actual post data details: %v", err)
	}
	if !reflect.DeepEqual(expectedCommonData, gotCommonData) {
		t.Errorf("Expected CommonPostData `%v`, but got `%v`", expectedCommonPostDataDetails, postDataCommon)
	}

	select {
	case msg := <-configChannel:
		if msg.MsgType != expectedMessage.MsgType {
			t.Errorf("expected MsgType %+v, but got %+v", expectedMessage.MsgType, msg.MsgType)
		}
		if msg.MsgMethod != expectedMessage.MsgMethod {
			t.Errorf("expected MsgMethod %+v, but got %+v", expectedMessage.MsgMethod, msg.MsgMethod)
		}
		if !reflect.DeepEqual(expectedMessage.AuthSubData, msg.AuthSubData) {
			t.Errorf("expected AuthSubData %+v, but got %+v", expectedMessage.AuthSubData, msg.AuthSubData)
		}
		if expectedMessage.Imsi != msg.Imsi {
			t.Errorf("expected IMSI %+v, but got %+v", expectedMessage.Imsi, msg.Imsi)
		}
	default:
		t.Error("expected message in configChannel, but none received")
	}
}

func TestSubscriberPostHandlersSubscriberExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)

	postDataCommon := make([]map[string]interface{}, 0)
	route := "/api/subscriber/imsi-208930100007487"
	inputData := map[string]string{
		"plmnID":         "12345",
		"opc":            "8e27b6af0e692e750f32667a3b14605d",
		"key":            "8baf473f2f8fd09487cccbd7097c6862",
		"sequenceNumber": "16f3b3f70fc2",
	}

	jsonData, err := json.Marshal(inputData)
	if err != nil {
		t.Fatalf("failed to marshal input data to JSON: %v", err)
	}

	commonDbAdapter := &MockMongoClientOneSubscriber{PostDataCommon: &postDataCommon}
	origDBClient := dbadapter.CommonDBClient
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 1)
	defer func() { configChannel = origChannel; dbadapter.CommonDBClient = origDBClient }()
	dbadapter.CommonDBClient = commonDbAdapter

	expectedCode := http.StatusConflict
	expectedBody := "subscriber imsi-208930100007487 already exists"
	expectedCommonPostDataDetails := []map[string]interface{}{
		{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]interface{}{"ueId": "imsi-208930100007487"}},
	}

	req, err := http.NewRequest(http.MethodPost, route, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != expectedCode {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if !strings.Contains(w.Body.String(), expectedBody) {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}

	expectedCommonData, err := json.Marshal(expectedCommonPostDataDetails)
	if err != nil {
		t.Fatalf("failed to marshal expected post data details: %v", err)
	}
	gotCommonData, err := json.Marshal(postDataCommon)
	if err != nil {
		t.Fatalf("failed to marshal actual post data details: %v", err)
	}
	if !reflect.DeepEqual(expectedCommonData, gotCommonData) {
		t.Errorf("Expected CommonPostData `%v`, but got `%v`", expectedCommonPostDataDetails, postDataCommon)
	}

	select {
	case <-configChannel:
		t.Error("expected no message in configChannel, but got one")
	default:
		// No message received as expected.
	}
}

type MockMongoClientAuthDB struct {
	MockMongoClientEmptyDB
}

func TestSubscriberDeleteSuccessNoDeviceGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	origDBClient := dbadapter.CommonDBClient
	dbAdapter := &MockMongoClientEmptyDB{}
	dbadapter.CommonDBClient = dbAdapter
	origAuthDBClient := dbadapter.AuthDBClient
	authDbAdapter := &MockMongoClientAuthDB{}
	dbadapter.AuthDBClient = authDbAdapter

	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusNoContent
	expectedBody := ""
	expectedMessage := configmodels.ConfigMessage{
		MsgType:   configmodels.Sub_data,
		MsgMethod: configmodels.Delete_op,
		Imsi:      "imsi-208930100007487",
	}
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 3)
	defer func() {
		configChannel = origChannel
		dbadapter.CommonDBClient = origDBClient
		dbadapter.AuthDBClient = origAuthDBClient
	}()
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
	if !strings.Contains(w.Body.String(), expectedBody) {
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
	select {
	case msg := <-configChannel:
		t.Errorf("expected no message in configChannel, but got %+v", msg)
	default:
	}
}

func TestSubscriberDeleteFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	origDBClient := dbadapter.CommonDBClient
	dbAdapter := &MockMongoClientDBError{}
	dbadapter.CommonDBClient = dbAdapter
	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusInternalServerError
	expectedBody := "error deleting subscriber. Please check the log for details"

	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 1)
	defer func() { configChannel = origChannel; dbadapter.CommonDBClient = origDBClient }()
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
	if !strings.Contains(w.Body.String(), expectedBody) {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}
	select {
	case msg := <-configChannel:
		t.Errorf("expected no message in configChannel, but got %+v", msg)
	default:
	}
}

func TestSubscriberDeleteSuccessWithDeviceGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	origDBClient := dbadapter.CommonDBClient
	dbAdapter := &MockMongoClientDeviceGroupsWithSubscriber{}
	dbadapter.CommonDBClient = dbAdapter
	origAuthDBClient := dbadapter.AuthDBClient
	authDbAdapter := &MockMongoClientAuthDB{}
	dbadapter.AuthDBClient = authDbAdapter
	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusNoContent
	expectedBody := ""
	expectedDeviceGroupMessage := configmodels.ConfigMessage{
		MsgType:      configmodels.Device_group,
		MsgMethod:    configmodels.Post_op,
		DevGroupName: "group1",
		DevGroup:     deviceGroupWithoutImsi(),
	}
	expectedMessage := configmodels.ConfigMessage{
		MsgType:   configmodels.Sub_data,
		MsgMethod: configmodels.Delete_op,
		Imsi:      "imsi-208930100007487",
	}
	origChannel := configChannel
	configChannel = make(chan *configmodels.ConfigMessage, 3)
	defer func() {
		configChannel = origChannel
		dbadapter.CommonDBClient = origDBClient
		dbadapter.AuthDBClient = origAuthDBClient
	}()
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if expectedCode != w.Code {
		t.Errorf("Expected status code `%v`, got `%v`", expectedCode, w.Code)
	}
	if expectedBody != w.Body.String() {
		t.Errorf("Expected body `%v`, got `%v`", expectedBody, w.Body.String())
	}
	select {
	case msg := <-configChannel:
		if expectedDeviceGroupMessage.MsgType != msg.MsgType {
			t.Errorf("expected MsgType %+v, but got %+v", expectedDeviceGroupMessage.MsgType, msg.MsgType)
		}
		if expectedDeviceGroupMessage.MsgMethod != msg.MsgMethod {
			t.Errorf("expected MsgMethod %+v, but got %+v", expectedDeviceGroupMessage.MsgMethod, msg.MsgMethod)
		}
		if expectedDeviceGroupMessage.DevGroupName != msg.DevGroupName {
			t.Errorf("expected device group name %+v, but got %+v", expectedDeviceGroupMessage.DevGroupName, msg.DevGroupName)
		}
		if !reflect.DeepEqual(expectedDeviceGroupMessage.DevGroup.Imsis, msg.DevGroup.Imsis) {
			t.Errorf("expected IMSIs in device group: %+v, but got %+v", expectedDeviceGroupMessage.DevGroup.Imsis, msg.DevGroup.Imsis)
		}
	default:
		t.Error("expected message in configChannel, but none received")
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
	select {
	case msg := <-configChannel:
		t.Errorf("expected no message in configChannel, but got %+v", msg)
	default:
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
