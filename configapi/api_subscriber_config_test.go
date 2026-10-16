// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	subscriberByImsiPath = "/api/subscriber/imsi-208930100007487"
	sstKey               = "sst"
	sampleImsi           = "imsi-2089300007487"
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
	subscriber[ueIdKey] = "208930100007487"
	subscriber[servingPlmnIdKey] = "12345"
	results = append(results, subscriber)
	return results, nil
}

func (m *MockMongoClientOneSubscriber) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		collKey:   collName,
		filterKey: filter,
	})
	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber[ueIdKey] = "208930100007487"
	subscriber[servingPlmnIdKey] = "12345"
	return subscriber, nil
}

func (m *MockMongoClientManySubscribers) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	ueIds := []string{"208930100007487", "208930100007488"}
	plmnIDs := []string{"12345", "54321"}
	for i, ueId := range ueIds {
		subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
		subscriber[ueIdKey] = ueId
		subscriber[servingPlmnIdKey] = plmnIDs[i]
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
		collKey:   coll,
		filterKey: filter,
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
		collKey:   coll,
		filterKey: filter,
	})
	sqn := models.NewSequenceNumber()
	sqn.SetSqn("16f3b3f70fc2")
	var authenticationSubscription models.AuthenticationSubscription
	authenticationSubscription.SetAuthenticationManagementField("8000")
	authenticationSubscription.SetAuthenticationMethod(models.AUTHMETHOD__5_G_AKA)
	authenticationSubscription.SetEncOpcKey("981d464c7c52eb6e5036234984ad0bcf")
	authenticationSubscription.SetEncPermanentKey("5122250214c33e723a5dd523fc145fc0")
	authenticationSubscription.SetSequenceNumber(*sqn)
	authSubscription := configmodels.ToBsonM(authenticationSubscription)
	if authSubscription == nil {
		logger.DbLog.Fatalln("failed to convert subscriber to BsonM")
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
		collKey:   coll,
		filterKey: filter,
	})
	return nil, nil
}

func (m *MockCommonDBClientEmpty) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		collKey:   coll,
		filterKey: filter,
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
		collKey:   coll,
		filterKey: filter,
	})

	switch coll {
	case amDataColl:
		nssais := models.NewNssaiWithDefaults()
		var snssai models.Snssai
		snssai.SetSst(1)
		snssai.SetSd("010203")
		nssais.SetDefaultSingleNssais([]models.Snssai{snssai})
		nssais.SetSingleNssais([]models.Snssai{snssai})

		var amData models.AccessAndMobilitySubscriptionData
		amData.SetGpsis([]string{sampleMsisdn})
		amData.SetNssai(*nssais)
		amData.SetSubscribedUeAmbr(*models.NewAmbr(ambr1000Kbps, ambr1000Kbps))
		amDataData := configmodels.ToBsonM(amData)
		if amDataData == nil {
			logger.DbLog.Fatalln("failed to convert amDataData to BsonM")
		}
		amDataData[servingPlmnIdKey] = "12345"
		return amDataData, nil

	case amPolicyDataColl:
		var amPolData models.AmPolicyData
		amPolData.SetSubscCats([]string{subscCatAether})
		amPolicyData := configmodels.ToBsonM(amPolData)
		if amPolicyData == nil {
			logger.DbLog.Fatalln("failed to convert amPolicyData to BsonM")
		}
		return amPolicyData, nil

	case smPolicyDataColl:
		var snssai models.Snssai
		snssai.SetSst(1)
		snssai.SetSd("010203")
		var smPolDnnData models.SmPolicyDnnData
		smPolDnnData.SetDnn(dnnInternet)
		var smPolSnssaiData models.SmPolicySnssaiData
		smPolSnssaiData.SetSnssai(snssai)
		smPolSnssaiData.SetSmPolicyDnnData(map[string]models.SmPolicyDnnData{dnnInternet: smPolDnnData})
		var smPolData models.SmPolicyData
		smPolData.SetSmPolicySnssaiData(map[string]models.SmPolicySnssaiData{"01010203": smPolSnssaiData})

		smPolicyData := configmodels.ToBsonM(smPolData)
		if smPolicyData == nil {
			logger.DbLog.Fatalln("failed to convert smPolicyData to BsonM")
		}
		return smPolicyData, nil

	case smfSelDataColl:
		var dnnInfo models.DnnInfo
		dnnInfo.SetDnn(models.AccessAndMobilitySubscriptionDataSubscribedDnnListInner{String: new(dnnInternet)})
		var snssaiInfo models.SnssaiInfo
		snssaiInfo.SetDnnInfos([]models.DnnInfo{dnnInfo})

		var smfSelSubsData models.SmfSelectionSubscriptionData
		smfSelSubsData.SetSubscribedSnssaiInfos(map[string]models.SnssaiInfo{"01010203": snssaiInfo})

		smfSelData := configmodels.ToBsonM(smfSelSubsData)
		if smfSelData == nil {
			logger.DbLog.Fatalln("failed to convert smfSelData to BsonM")
		}
		return smfSelData, nil

	default:
		return nil, fmt.Errorf("collection %s not found", coll)
	}
}

func (m *MockCommonDBClientWithData) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	m.postDataCommon = append(m.postDataCommon, map[string]any{
		collKey:   coll,
		filterKey: filter,
	})

	arp := models.NewArpWithDefaults()
	arp.SetPriorityLevel(8)
	arp.SetPreemptCap(models.PREEMPTIONCAPABILITY_MAY_PREEMPT)
	arp.SetPreemptVuln(models.PREEMPTIONVULNERABILITY_PREEMPTABLE)
	var5gQosProfile := models.NewSubscribedDefaultQos(9, *arp)
	var5gQosProfile.SetPriorityLevel(8)
	smDataData := []models.SessionManagementSubscriptionData{
		{
			SingleNssai: models.Snssai{
				Sst: 1,
				Sd:  new("010203"),
			},
			DnnConfigurations: &map[string]models.DnnConfiguration{
				dnnInternet: {
					PduSessionTypes: models.PduSessionTypes{
						DefaultSessionType:  models.PDUSESSIONTYPE_IPV4.Ptr(),
						AllowedSessionTypes: []models.PduSessionType{models.PDUSESSIONTYPE_IPV4},
					},
					SscModes: models.SscModes{
						DefaultSscMode:  models.SSCMODE_SSC_MODE_1,
						AllowedSscModes: []models.SscMode{models.SSCMODE_SSC_MODE_1},
					},
					SessionAmbr:     models.NewAmbr(ambr1000Kbps, ambr1000Kbps),
					Var5gQosProfile: var5gQosProfile,
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
			ueId:                 sampleImsi,
			route:                "/api/subscriber/:ueId",
			commonDbAdapter:      &MockCommonDBClientEmpty{},
			authDbAdapter:        &MockAuthDBClientEmpty{},
			expectedHTTPStatus:   http.StatusNotFound,
			expectedFullResponse: map[string]any{errorKey: "subscriber with ID imsi-2089300007487 not found"},
			expectedCommonPostDataDetails: []map[string]any{
				{collKey: amDataColl, filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "subscriptionData.provisionedData.smData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "subscriptionData.provisionedData.smfSelectionSubscriptionData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "policyData.ues.amData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "policyData.ues.smData", filterKey: map[string]any{ueIdKey: sampleImsi}},
			},
			expectedAuthPostDataDetails: []map[string]any{
				{collKey: "subscriptionData.authenticationData.authenticationSubscription", filterKey: map[string]any{ueIdKey: sampleImsi}},
			},
		},

		{
			name:               "Valid subscriber data retrieved",
			ueId:               sampleImsi,
			commonDbAdapter:    &MockCommonDBClientWithData{},
			authDbAdapter:      &MockAuthDBClientWithData{},
			route:              "/api/subscriber/:ueId",
			expectedHTTPStatus: http.StatusOK,
			expectedFullResponse: map[string]any{
				"AccessAndMobilitySubscriptionData": map[string]any{
					"activeTime":          nil,
					"ecRestrictionDataWb": nil,
					"gpsis":               []any{sampleMsisdn},
					"nssai": map[string]any{
						"defaultSingleNssais": []any{
							map[string]any{"sd": "010203", sstKey: 1},
						},
						"singleNssais": []any{
							map[string]any{"sd": "010203", sstKey: 1},
						},
					},
					"odbPacketServices": nil,
					"rfspIndex":         nil,
					"subsRegTimer":      nil,
					"subscribedUeAmbr": map[string]any{
						downlinkKey: ambr1000Kbps,
						uplinkKey:   ambr1000Kbps,
					},
					"traceData": nil,
				},
				"AmPolicyData": map[string]any{
					"subscCats": []any{subscCatAether},
				},
				"AuthenticationSubscription": map[string]any{
					"authenticationManagementField": "8000",
					"authenticationMethod":          models.AUTHMETHOD__5_G_AKA,
					"encOpcKey":                     "981d464c7c52eb6e5036234984ad0bcf",
					"encPermanentKey":               "5122250214c33e723a5dd523fc145fc0",
					"sequenceNumber":                map[string]any{"sqn": "16f3b3f70fc2"},
				},
				"FlowRules": nil,
				"SessionManagementSubscriptionData": []any{
					map[string]any{
						"dnnConfigurations": map[string]any{
							dnnInternet: map[string]any{
								"5gQosProfile": map[string]any{
									"5qi":            9,
									"arp":            map[string]any{"preemptCap": "MAY_PREEMPT", "preemptVuln": "PREEMPTABLE", priorityLevelKey: 8},
									priorityLevelKey: 8,
								},
								"dnAaaAddress":      nil,
								"ecsAddrConfigInfo": nil,
								"pduSessionTypes": map[string]any{
									"allowedSessionTypes": []any{"IPV4"},
									"defaultSessionType":  "IPV4",
								},
								"sessionAmbr": map[string]any{
									downlinkKey: ambr1000Kbps,
									uplinkKey:   ambr1000Kbps,
								},
								"sscModes": map[string]any{
									"allowedSscModes": []any{"SSC_MODE_1"},
									"defaultSscMode":  "SSC_MODE_1",
								},
							},
						},
						"odbPacketServices": nil,
						"singleNssai": map[string]any{
							"sd":   "010203",
							sstKey: 1,
						},
						"traceData": nil,
					},
				},
				"SmPolicyData": map[string]any{
					"smPolicySnssaiData": map[string]any{
						"01010203": map[string]any{
							"smPolicyDnnData": map[string]any{
								dnnInternet: map[string]any{
									"dnn": dnnInternet,
								},
							},
							"snssai": map[string]any{
								"sd":   "010203",
								sstKey: 1,
							},
						},
					},
				},
				"SmfSelectionSubscriptionData": map[string]any{
					"subscribedSnssaiInfos": map[string]any{
						"01010203": map[string]any{
							"dnnInfos": []any{
								map[string]any{
									"dnn": dnnInternet,
								},
							},
						},
					},
				},
				"plmnID": "12345",
				ueIdKey:  sampleImsi,
			},
			expectedCommonPostDataDetails: []map[string]any{
				{collKey: amDataColl, filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "subscriptionData.provisionedData.smData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "subscriptionData.provisionedData.smfSelectionSubscriptionData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "policyData.ues.amData", filterKey: map[string]any{ueIdKey: sampleImsi}},
				{collKey: "policyData.ues.smData", filterKey: map[string]any{ueIdKey: sampleImsi}},
			},
			expectedAuthPostDataDetails: []map[string]any{
				{collKey: "subscriptionData.authenticationData.authenticationSubscription", filterKey: map[string]any{ueIdKey: sampleImsi}},
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

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("/api/subscriber/%s", tt.ueId), nil)
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
			route:        subscriberPath,
			dbAdapter:    &MockCommonDBClientEmpty{},
			expectedCode: http.StatusOK,
			expectedBody: "[]",
		},
		{
			name:         "Get subscribers list with one element",
			route:        subscriberPath,
			dbAdapter:    &MockMongoClientOneSubscriber{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"plmnID":"12345","ueId":"208930100007487"}]`,
		},
		{
			name:         "ManySubscribers",
			route:        subscriberPath,
			dbAdapter:    &MockMongoClientManySubscribers{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"plmnID":"12345","ueId":"208930100007487"},{"plmnID":"54321","ueId":"208930100007488"}]`,
		},
		{
			name:         "SubscriberDBError",
			route:        subscriberPath,
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
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, tc.route, nil)
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
	subscriber := configmodels.ToBsonM(authenticationSubscription())
	subscriber[ueIdKey] = db.subscribers[0]
	subscriber[servingPlmnIdKey] = "12345"
	return subscriber, nil
}

func (db *AuthDBMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.receivedPostData = append(db.receivedPostData, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	})
	return true, nil
}

func (db *AuthDBMockDBClient) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	params := map[string]any{
		collKey:   collName,
		filterKey: filter,
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
		collKey:   collName,
		filterKey: filter,
	})

	if db.err != nil {
		return nil, db.err
	}
	if len(db.subscribers) == 0 {
		return nil, nil
	}

	subscriber := configmodels.ToBsonM(models.AccessAndMobilitySubscriptionData{})
	subscriber[ueIdKey] = db.subscribers[0]
	subscriber[servingPlmnIdKey] = "12345"
	return subscriber, nil
}

func (db *PostSubscriberMockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	db.receivedPostData = append(db.receivedPostData, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	})
	return true, nil
}

func (db *PostSubscriberMockDBClient) StartSession() (dbadapter.DBSession, error) {
	return &MockSession{}, nil
}

func TestSubscriberPost(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

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
				subscribers: []string{testSubscriberImsi},
			},
			expectedCode: http.StatusConflict,
			expectedBody: "subscriber imsi-208930100007487 already exists",
			expectedGetData: []map[string]any{
				{collKey: amDataColl, filterKey: map[string]any{ueIdKey: testSubscriberImsi}},
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
				{collKey: amDataColl, filterKey: map[string]any{ueIdKey: testSubscriberImsi}},
			},
			expectedPostData: []map[string]any{
				{collKey: amDataColl, filterKey: bson.M{ueIdKey: testSubscriberImsi}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.Default()
			AddApiService(router)

			route := subscriberByImsiPath
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
			authMock := &AuthDBMockDBClient{}
			dbadapter.AuthDBClient = authMock
			dbadapter.CommonDBClient = &tc.commonDbAdapter

			expectedGetData, err := json.Marshal(tc.expectedGetData)
			if err != nil {
				t.Fatalf("failed to marshal expected get data details: %v", err)
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, route, bytes.NewBuffer(jsonData))
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
				if len(authMock.receivedPostData) != 1 {
					t.Fatalf("expected 1 auth DB post call, got %d", len(authMock.receivedPostData))
				}
				if authMock.receivedPostData[0][collKey] != authSubsDataColl {
					t.Errorf("expected auth collection %v, got %v", authSubsDataColl, authMock.receivedPostData[0][collKey])
				}
				expectedAmDataCollection := amDataColl
				if len(tc.commonDbAdapter.receivedPostData) != 1 {
					t.Fatalf("expected 1 amData post call, got %d", len(tc.commonDbAdapter.receivedPostData))
				}
				if tc.commonDbAdapter.receivedPostData[0][collKey] != expectedAmDataCollection {
					t.Errorf("expected collection %v, got %v", expectedAmDataCollection, tc.commonDbAdapter.receivedPostData[0][collKey])
				}
				expectedFilter := bson.M{ueIdKey: testSubscriberImsi}
				if !reflect.DeepEqual(tc.commonDbAdapter.receivedPostData[0][filterKey], expectedFilter) {
					t.Errorf("expected filter %v, got %v", expectedFilter, tc.commonDbAdapter.receivedPostData[0][filterKey])
				}
			}
		})
	}
}

type DeleteSubscriberMockDBClient struct {
	dbadapter.DBInterface
	deviceGroups      []configmodels.DeviceGroups
	deleteData        []map[string]any
	deleteWithCtxData []map[string]any
	err               error
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	if coll == "device_group" {
		dg := configmodels.ToBsonM(db.deviceGroups[0])
		if dg == nil {
			logger.DbLog.Fatalln("failed to convert device group to BsonM")
		}
		return dg, nil
	}
	return nil, nil
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if db.err != nil {
		return nil, db.err
	}
	// Other collections (e.g. sliceDataColl, looked up when syncing a device group's associated
	// slice) must not be answered with device group documents, which cannot unmarshal as anything
	// else.
	if coll != devGroupDataColl {
		return nil, nil
	}
	var results []map[string]any
	for _, deviceGroup := range db.deviceGroups {
		dg := configmodels.ToBsonM(deviceGroup)
		if dg == nil {
			logger.DbLog.Fatalln("failed to convert device groups to BsonM")
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
		collKey:   coll,
		filterKey: filter,
	}
	db.deleteData = append(db.deleteData, params)
	return nil
}

func (db *DeleteSubscriberMockDBClient) StartSession() (dbadapter.DBSession, error) {
	return &MockSession{}, nil
}

func (db *DeleteSubscriberMockDBClient) RestfulAPIDeleteOneWithContext(ctx context.Context, collName string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	db.deleteWithCtxData = append(db.deleteWithCtxData, map[string]any{
		collKey:   collName,
		filterKey: filter,
	})
	return nil
}

func TestSubscriberDelete(t *testing.T) {
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	tests := []struct {
		name            string
		commonDbAdapter dbadapter.DBInterface
		expectedCode    int
	}{
		{
			name: "Subscriber belongs to a device group",
			commonDbAdapter: &DeleteSubscriberMockDBClient{
				deviceGroups: []configmodels.DeviceGroups{
					deviceGroupWithImsis(testGroupName, []string{"208930100007487"}),
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
			route := subscriberByImsiPath
			expectedCode := tc.expectedCode
			expectedBody := ""

			req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, route, nil)
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
	cleanupFactory := setupTestFactory()
	defer cleanupFactory()

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddApiService(router)
	origDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origDBClient }()
	dbadapter.CommonDBClient = &DeleteSubscriberMockDBClient{
		err: fmt.Errorf("mock error"),
	}
	route := subscriberByImsiPath
	expectedCode := http.StatusInternalServerError
	expectedBody := "error deleting subscriber. Please check the log for details"

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, route, nil)
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
		BitrateUnit:    bitrateUnitKbps,
		TrafficClass:   &trafficClass,
	}
	ipDomain := configmodels.DeviceGroupsIpDomainExpanded{
		Dnn:          dnnInternet,
		UeIpPool:     "172.250.1.0/16",
		DnsPrimary:   "1.1.1.1",
		DnsSecondary: "8.8.8.8",
		Mtu:          1460,
		UeDnnQos:     &qos,
	}
	deviceGroup := configmodels.DeviceGroups{
		DeviceGroupName: name,
		Imsis:           imsis,
		SiteInfo:        demoSiteName,
		IpDomainName:    "pool1",
		IpDomainsExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			ipDomain,
		},
	}
	return deviceGroup
}
