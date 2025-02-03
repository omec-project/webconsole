// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		expectedCode 	int
		expectedBody string
		expectedMessage configmodels.ConfigMessage
	}{
		{
			name:      "Create a new subscriber success",
			route:     "/api/subscriber/imsi-208930100007487",
			inputData: `{"UeId":"208930100007487", "plmnId":"12345", "opc":"981d464c7c52eb6e5036234984ad0bcf","key":"5122250214c33e723a5dd523fc145fc0", "sequenceNumber":"16f3b3f70fc2"}`,
			expectedMessage: configmodels.ConfigMessage{
				MsgType:   configmodels.Sub_data,
				MsgMethod: configmodels.Post_op,
				AuthSubData: &models.AuthenticationSubscription{
					AuthenticationManagementField: "8000",
					AuthenticationMethod:          "5G_AKA", // "5G_AKA", "EAP_AKA_PRIME"
					Milenage: &models.Milenage{
						Op: &models.Op{
							EncryptionAlgorithm: 0,
							EncryptionKey:       0,
							OpValue:             "", // Required
						},
					},
					Opc: &models.Opc{
						EncryptionAlgorithm: 0,
						EncryptionKey:       0,
						// OpcValue:            "8e27b6af0e692e750f32667a3b14605d", // Required
					},
					PermanentKey: &models.PermanentKey{
						EncryptionAlgorithm: 0,
						EncryptionKey:       0,
						// PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862", // Required
					},
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
				if tc.expectedMessage.AuthSubData != nil {
					if msg.AuthSubData == nil {
						t.Errorf("expected AuthSubData %+v, but got nil", tc.expectedMessage.AuthSubData)
					}
					if tc.expectedMessage.Imsi != msg.Imsi {
						t.Errorf("expected IMSI %+v, but got %+v", tc.expectedMessage.Imsi, msg.Imsi)
					}
				}
			default:
				t.Error("expected message in configChannel, but none received")
			}
		})
	}
}

func TestSubscriberDeleteHandlers(t *testing.T) {
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
			name:         "Delete a subscriber success",
			route:        "/api/subscriber/imsi-208930100007487",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusNoContent,
			expectedBody: "",
		},
		{
			name:         "Delete subscriber DB Failure",
			route:        "/api/subscriber/imsi-208930100007487",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to delete subscriber"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodDelete, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if tc.expectedBody != w.Body.String() {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}
