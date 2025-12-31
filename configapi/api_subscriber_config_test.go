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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type PostDataTracker interface {
	dbadapter.DBInterface
	GetPostData() []map[string]any
}

type MockMongoClientOneSubscriber struct {
	dbadapter.DBInterface
	postDataCommon []map[string]any
}

type MockMongoClientManySubscribers struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientOneSubscriber) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber["ueId"] = "208930100007487"
	subscriber["servingPlmnId"] = "12345"
	results = append(results, subscriber)
	return results, nil
}

func (m *MockMongoClientOneSubscriber) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		"coll":   collName,
		"filter": filter,
	})
	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber["ueId"] = "208930100007487"
	subscriber["servingPlmnId"] = "12345"
	return subscriber, nil
}

func (m *MockMongoClientManySubscribers) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
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

type MockAuthDBClientEmpty struct {
	dbadapter.DBInterface
	postDataAuth []map[string]any
}

func (m *MockAuthDBClientEmpty) GetPostData() []map[string]any {
	return m.postDataAuth
}

func (m *MockAuthDBClientEmpty) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	m.postDataAuth = append(m.postDataAuth, map[string]any{
		"coll":   coll,
		"filter": filter,
	})
	return nil, nil
}

type MockAuthDBClientWithData struct {
	dbadapter.DBInterface
	postDataAuth []map[string]any
}

func (m *MockAuthDBClientWithData) GetPostData() []map[string]any {
	return m.postDataAuth
}

func (m *MockAuthDBClientWithData) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	m.postDataAuth = append(m.postDataAuth, map[string]any{
		"coll":   coll,
		"filter": filter,
	})

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
			EncryptionKey:       "",
			PermanentKeyValue:   "5122250214c33e723a5dd523fc145fc0",
		},
		SequenceNumber: "16f3b3f70fc2",
	})
	if authSubscription == nil {
		logger.AppLog.Fatalln("failed to convert subscriber to BsonM")
	}
	return authSubscription, nil
}

type MockCommonDBClientEmpty struct {
	dbadapter.DBInterface
	postDataCommon []map[string]any
}

func (m *MockCommonDBClientEmpty) GetPostData() []map[string]any {
	return m.postDataCommon
}

func (m *MockCommonDBClientEmpty) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		"coll":   coll,
		"filter": filter,
	})
	return nil, nil
}

func (m *MockCommonDBClientEmpty) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		"coll":   coll,
		"filter": filter,
	})
	return nil, nil
}

type MockCommonDBClientWithData struct {
	dbadapter.DBInterface
	postDataCommon []map[string]any
}

func (m *MockCommonDBClientWithData) GetPostData() []map[string]any {
	return m.postDataCommon
}

func (m *MockCommonDBClientWithData) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		"coll":   coll,
		"filter": filter,
	})

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
			logger.AppLog.Fatalln("failed to convert amDataData to BsonM")
		}
		return amDataData, nil

	case "policyData.ues.amData":
		amPolicyData := configmodels.ToBsonM(models.AmPolicyData{
			SubscCats: []string{
				"aether",
			},
		})
		if amPolicyData == nil {
			logger.AppLog.Fatalln("failed to convert amPolicyData to BsonM")
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
			logger.AppLog.Fatalln("failed to convert smPolicyData to BsonM")
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
			logger.AppLog.Fatalln("failed to convert smfSelData to BsonM")
		}
		return smfSelData, nil

	default:
		return nil, fmt.Errorf("collection %s not found", coll)
	}
}

func (m *MockCommonDBClientWithData) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		"coll":   coll,
		"filter": filter,
	})

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
	result := make([]map[string]any, len(smDataData))
	for i, smData := range smDataData {
		result[i] = map[string]any{
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

	tests := []struct {
		name                          string
		ueId                          string
		route                         string
		commonDbAdapter               PostDataTracker
		authDbAdapter                 PostDataTracker
		expectedHTTPStatus            int
		expectedFullResponse          map[string]any
		expectedCommonPostDataDetails []map[string]any
		expectedAuthPostDataDetails   []map[string]any
	}{
		{
			name:                 "No subscriber data found",
			ueId:                 "imsi-2089300007487",
			route:                "/api/subscriber/:ueId",
			commonDbAdapter:      &MockCommonDBClientEmpty{},
			authDbAdapter:        &MockAuthDBClientEmpty{},
			expectedHTTPStatus:   http.StatusNotFound,
			expectedFullResponse: map[string]any{"error": "subscriber with ID imsi-2089300007487 not found"},
			expectedCommonPostDataDetails: []map[string]any{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smfSelectionSubscriptionData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.amData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.smData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
			},
			expectedAuthPostDataDetails: []map[string]any{
				{"coll": "subscriptionData.authenticationData.authenticationSubscription", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
			},
		},

		{
			name:               "Valid subscriber data retrieved",
			ueId:               "imsi-2089300007487",
			commonDbAdapter:    &MockCommonDBClientWithData{},
			authDbAdapter:      &MockAuthDBClientWithData{},
			route:              "/api/subscriber/:ueId",
			expectedHTTPStatus: http.StatusOK,
			expectedFullResponse: map[string]any{
				"AccessAndMobilitySubscriptionData": map[string]any{
					"gpsis": []any{"msisdn-0900000000"},
					"nssai": map[string]any{
						"defaultSingleNssais": []any{
							map[string]any{"sd": "010203", "sst": 1},
						},
						"singleNssais": []any{
							map[string]any{"sd": "010203", "sst": 1},
						},
					},
					"subscribedUeAmbr": map[string]any{
						"downlink": "1000 Kbps",
						"uplink":   "1000 Kbps",
					},
				},
				"AmPolicyData": map[string]any{
					"subscCats": []any{"aether"},
				},
				"AuthenticationSubscription": map[string]any{
					"authenticationManagementField": "8000",
					"authenticationMethod":          "5G_AKA",
					"milenage": map[string]any{
						"op": map[string]any{
							"encryptionAlgorithm": 0,
							"encryptionKey":       0,
							"opValue":             "c9e8763286b5b9ffbdf56e1297d0887b",
						},
					},
					"opc": map[string]any{
						"encryptionAlgorithm": 0,
						"encryptionKey":       0,
						"opcValue":            "981d464c7c52eb6e5036234984ad0bcf",
					},
					"permanentKey": map[string]any{
						"encryptionAlgorithm": 0,
						"encryptionKey":       "",
						"permanentKeyValue":   "5122250214c33e723a5dd523fc145fc0",
					},
					"sequenceNumber": "16f3b3f70fc2",
				},
				"FlowRules": nil,
				"SessionManagementSubscriptionData": []any{
					map[string]any{
						"dnnConfigurations": map[string]any{
							"internet": map[string]any{
								"5gQosProfile": map[string]any{
									"5qi":           9,
									"arp":           map[string]any{"preemptCap": "", "preemptVuln": "", "priorityLevel": 8},
									"priorityLevel": 8,
								},
								"pduSessionTypes": map[string]any{
									"allowedSessionTypes": []any{"IPV4"},
									"defaultSessionType":  "IPV4",
								},
								"sessionAmbr": map[string]any{
									"downlink": "1000 Kbps",
									"uplink":   "1000 Kbps",
								},
								"sscModes": map[string]any{
									"allowedSscModes": []any{"SSC_MODE_1"},
									"defaultSscMode":  "SSC_MODE_1",
								},
							},
						},
						"singleNssai": map[string]any{
							"sd":  "010203",
							"sst": 1,
						},
					},
				},
				"SmPolicyData": map[string]any{
					"smPolicySnssaiData": map[string]any{
						"01010203": map[string]any{
							"smPolicyDnnData": map[string]any{
								"internet": map[string]any{
									"dnn": "internet",
								},
							},
							"snssai": map[string]any{
								"sd":  "010203",
								"sst": 1,
							},
						},
					},
				},
				"SmfSelectionSubscriptionData": map[string]any{
					"subscribedSnssaiInfos": map[string]any{
						"01010203": map[string]any{
							"dnnInfos": []any{
								map[string]any{
									"dnn": "internet",
								},
							},
						},
					},
				},
				"plmnID": "",
				"ueId":   "imsi-2089300007487",
			},
			expectedCommonPostDataDetails: []map[string]any{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "subscriptionData.provisionedData.smfSelectionSubscriptionData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.amData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
				{"coll": "policyData.ues.smData", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
			},
			expectedAuthPostDataDetails: []map[string]any{
				{"coll": "subscriptionData.authenticationData.authenticationSubscription", "filter": map[string]any{"ueId": "imsi-2089300007487"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				t.Errorf("expected HTTP status %d, got %d", tt.expectedHTTPStatus, w.Code)
			}

			responseContent := w.Body.String()
			var actual map[string]any
			if err := json.Unmarshal([]byte(responseContent), &actual); err != nil {
				t.Fatalf("failed to unmarshal actual response: %v. Raw response: %s", err, responseContent)
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
				t.Errorf("mismatch in response:\nExpected:\n%s\nGot:\n%s\n", string(expectedResponse), string(actualResponse))
			}

			expectedCommonData, err := json.Marshal(tt.expectedCommonPostDataDetails)
			if err != nil {
				t.Fatalf("failed to marshal expected post data details: %v", err)
			}
			gotCommonData, err := json.Marshal(tt.commonDbAdapter.GetPostData())
			if err != nil {
				t.Fatalf("failed to marshal actual post data details: %v", err)
			}
			if !reflect.DeepEqual(expectedCommonData, gotCommonData) {
				t.Errorf("expected CommonPostData `%v`, but got `%v`", tt.expectedCommonPostDataDetails, gotCommonData)
			}

			expectedAuthData, err := json.Marshal(tt.expectedAuthPostDataDetails)
			if err != nil {
				t.Fatalf("failed to marshal expected auth post data details: %v", err)
			}
			gotAuthData, err := json.Marshal(tt.authDbAdapter.GetPostData())
			if err != nil {
				t.Fatalf("failed to marshal actual auth post data details: %v", err)
			}
			if !reflect.DeepEqual(expectedAuthData, gotAuthData) {
				t.Errorf("expected AuthPostData `%v`, but got `%v`", tt.expectedAuthPostDataDetails, gotAuthData)
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
			dbAdapter:    &MockCommonDBClientEmpty{},
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}

type AuthDBMockDBClient struct {
	dbadapter.DBInterface
	subscribers      []string
	receivedPostData []map[string]any
	deleteData       []map[string]any
	err              error
}

func (db *AuthDBMockDBClient) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	if len(db.subscribers) == 0 {
		return nil, nil
	}
	s := models.AuthenticationSubscription{
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
			EncryptionKey:       "",
			PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
		},
		SequenceNumber: "16f3b3f70fc2",
	}

	subscriber := configmodels.ToBsonM(s)
	subscriber["ueId"] = db.subscribers[0]
	subscriber["servingPlmnId"] = "12345"
	return subscriber, nil
}

func (db *AuthDBMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.receivedPostData = append(db.receivedPostData, map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	})
	return true, nil
}

func (db *AuthDBMockDBClient) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	params := map[string]any{
		"coll":   collName,
		"filter": filter,
	}
	db.deleteData = append(db.deleteData, params)
	return nil
}

type PostSubscriberMockDBClient struct {
	dbadapter.DBInterface
	subscribers      []string
	receivedGetData  []map[string]any
	receivedPostData []map[string]any
	err              error
}

func (db *PostSubscriberMockDBClient) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	db.receivedGetData = append(db.receivedGetData, map[string]any{
		"coll":   collName,
		"filter": filter,
	})

	if db.err != nil {
		return nil, db.err
	}
	if len(db.subscribers) == 0 {
		return nil, nil
	}

	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber["ueId"] = db.subscribers[0]
	subscriber["servingPlmnId"] = "12345"
	return subscriber, nil
}

func (db *PostSubscriberMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.receivedPostData = append(db.receivedPostData, map[string]any{
		"coll":   collName,
		"filter": filter,
		"data":   postData,
	})
	return true, nil
}

func TestSubscriberPost(t *testing.T) {
	tests := []struct {
		name             string
		commonDbAdapter  PostSubscriberMockDBClient
		expectedCode     int
		expectedBody     string
		expectedGetData  []map[string]any
		expectedPostData []map[string]any
	}{
		{
			name: "Existing subscriber is rejected",
			commonDbAdapter: PostSubscriberMockDBClient{
				subscribers: []string{"imsi-208930100007487"},
			},
			expectedCode: http.StatusConflict,
			expectedBody: "subscriber imsi-208930100007487 already exists",
			expectedGetData: []map[string]any{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]any{"ueId": "imsi-208930100007487"}},
			},
			expectedPostData: nil,
		},
		{
			name: "New subscriber is created",
			commonDbAdapter: PostSubscriberMockDBClient{
				subscribers: []string{},
			},
			expectedCode: http.StatusCreated,
			expectedBody: `{}`,
			expectedGetData: []map[string]any{
				{"coll": "subscriptionData.provisionedData.amData", "filter": map[string]any{"ueId": "imsi-208930100007487"}},
			},
			expectedPostData: []map[string]any{
				{"coll": "subscriptionData.provisionedData.amData", "filter": bson.M{"ueId": "imsi-208930100007487"}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.Default()
			AddApiService(router)

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

			origDBClient := dbadapter.CommonDBClient
			origAuthDBClient := dbadapter.AuthDBClient
			defer func() {
				dbadapter.CommonDBClient = origDBClient
				dbadapter.AuthDBClient = origAuthDBClient
			}()
			dbadapter.AuthDBClient = &AuthDBMockDBClient{}
			dbadapter.CommonDBClient = &tc.commonDbAdapter

			expectedGetData, err := json.Marshal(tc.expectedGetData)
			if err != nil {
				t.Fatalf("failed to marshal expected get data details: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, route, bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tc.expectedBody) {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}

			gotGetData, err := json.Marshal(tc.commonDbAdapter.receivedGetData)
			if err != nil {
				t.Fatalf("failed to marshal actual get data details: %v", err)
			}
			if !reflect.DeepEqual(expectedGetData, gotGetData) {
				t.Errorf("expected expectedGetData `%+v`, but got `%+v`", expectedGetData, tc.commonDbAdapter.receivedGetData)
			}

			if tc.expectedPostData != nil {
				expectedAmDataCollection := AmDataColl
				if tc.commonDbAdapter.receivedPostData[0]["coll"] != expectedAmDataCollection {
					t.Errorf("expected collection %v, got %v", expectedAmDataCollection, tc.commonDbAdapter.receivedPostData[0]["coll"])
				}
				if !reflect.DeepEqual(tc.commonDbAdapter.receivedPostData[0]["filter"], tc.expectedPostData[0]["filter"]) {
					t.Errorf("expected filter %t, got %t", tc.expectedPostData[0]["filter"], tc.commonDbAdapter.receivedPostData[0]["filter"])
				}
				expectedFilter := bson.M{"ueId": "imsi-208930100007487"}
				if !reflect.DeepEqual(tc.commonDbAdapter.receivedPostData[0]["filter"], expectedFilter) {
					t.Errorf("expected filter %v, got %v", expectedFilter, tc.commonDbAdapter.receivedPostData[0]["filter"])
				}
			}
		})
	}
}

type DeleteSubscriberMockDBClient struct {
	dbadapter.DBInterface
	deviceGroups []configmodels.DeviceGroups
	deleteData   []map[string]any
	err          error
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if coll == "device_group" {
		dg := configmodels.ToBsonM(db.deviceGroups[0])
		if dg == nil {
			logger.AppLog.Fatalln("failed to convert device group to BsonM")
		}
		return dg, nil
	}
	return nil, nil
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, deviceGroup := range db.deviceGroups {
		dg := configmodels.ToBsonM(deviceGroup)
		if dg == nil {
			logger.AppLog.Fatalln("failed to convert device groups to BsonM")
		}
		results = append(results, dg)
	}
	return results, db.err
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIPost(coll string, filter bson.M, postData map[string]any) (bool, error) {
	if db.err != nil {
		return true, db.err
	}
	return true, nil
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIDeleteOne(coll string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	params := map[string]any{
		"coll":   coll,
		"filter": filter,
	}
	db.deleteData = append(db.deleteData, params)
	return nil
}

func TestSubscriberDelete(t *testing.T) {
	tests := []struct {
		name            string
		commonDbAdapter dbadapter.DBInterface
		expectedCode    int
	}{
		{
			name: "Subscriber belongs to a device group",
			commonDbAdapter: &DeleteSubscriberMockDBClient{
				deviceGroups: []configmodels.DeviceGroups{
					deviceGroupWithImsis("group1", []string{"208930100007487"}),
				},
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name: "Subscriber does not belongs to any device group",
			commonDbAdapter: &DeleteSubscriberMockDBClient{
				deviceGroups: []configmodels.DeviceGroups{},
			},
			expectedCode: http.StatusNoContent,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.Default()
			AddApiService(router)
			origDBClient := dbadapter.CommonDBClient
			origAuthDBClient := dbadapter.AuthDBClient
			defer func() {
				dbadapter.CommonDBClient = origDBClient
				dbadapter.AuthDBClient = origAuthDBClient
			}()
			dbadapter.CommonDBClient = tc.commonDbAdapter
			dbadapter.AuthDBClient = &AuthDBMockDBClient{}
			route := "/api/subscriber/imsi-208930100007487"
			expectedCode := tc.expectedCode
			expectedBody := ""

			req, err := http.NewRequest(http.MethodDelete, route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if expectedCode != w.Code {
				t.Errorf("expected status code `%v`, got `%v`", expectedCode, w.Code)
			}
			if expectedBody != w.Body.String() {
				t.Errorf("expected body `%v`, got `%v`", expectedBody, w.Body.String())
			}
		})
	}
}

func TestSubscriberDeleteFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	origDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origDBClient }()
	dbadapter.CommonDBClient = &DeleteSubscriberMockDBClient{
		err: fmt.Errorf("mock error"),
	}
	route := "/api/subscriber/imsi-208930100007487"
	expectedCode := http.StatusInternalServerError
	expectedBody := "error deleting subscriber. Please check the log for details"

	req, err := http.NewRequest(http.MethodDelete, route, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if expectedCode != w.Code {
		t.Errorf("expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if !strings.Contains(w.Body.String(), expectedBody) {
		t.Errorf("expected `%v`, got `%v`", expectedBody, w.Body.String())
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
