// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockMongoClientOneGnb struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientOneGnb) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	gnb := configmodels.Gnb{
		Name: "gnb1",
		Tac:  "123",
	}
	var gnbBson bson.M
	tmp, _ := json.Marshal(gnb)
	json.Unmarshal(tmp, &gnbBson)

	results = append(results, gnbBson)
	return results, nil
}

type MockMongoClientManyGnbs struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientManyGnbs) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	names := []string{"gnb0", "gnb1", "gnb2"}
	tacs := []string{"12", "345", "678"}
	for i, name := range names {
		gnb := configmodels.Gnb{
			Name: name,
			Tac:  tacs[i],
		}
		var gnbBson bson.M
		tmp, _ := json.Marshal(gnb)
		json.Unmarshal(tmp, &gnbBson)

		results = append(results, gnbBson)
	}
	return results, nil
}

type MockMongoClientOneUpf struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientOneUpf) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	upf := configmodels.Upf{
		Hostname: "upf1",
		Port:     "123",
	}
	var upfBson bson.M
	tmp, _ := json.Marshal(upf)
	json.Unmarshal(tmp, &upfBson)

	results = append(results, upfBson)
	return results, nil
}

func (m *MockMongoClientOneUpf) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

type MockMongoClientManyUpfs struct {
	dbadapter.DBInterface
}

func (m *MockMongoClientManyUpfs) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	names := []string{"upf0", "upf1", "upf2"}
	ports := []string{"12", "345", "678"}
	for i, name := range names {
		upf := configmodels.Upf{
			Hostname: name,
			Port:     ports[i],
		}
		var upfBson bson.M
		tmp, _ := json.Marshal(upf)
		json.Unmarshal(tmp, &upfBson)

		results = append(results, upfBson)
	}
	return results, nil
}

type MockMongoClientPutExistingUpf struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientPutExistingUpf) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (m *MockMongoClientPutExistingUpf) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

func (db *MockMongoClientPutExistingUpf) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) (bool, error) {
	return true, nil
}

func TestInventoryGetHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "GnbEmptyDB",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "[]",
		},
		{
			name:         "OneGnb",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientOneGnb{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"name":"gnb1","tac":"123"}]`,
		},
		{
			name:         "ManyGnbs",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientManyGnbs{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"name":"gnb0","tac":"12"},{"name":"gnb1","tac":"345"},{"name":"gnb2","tac":"678"}]`,
		},
		{
			name:         "GnbDBError",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to retrieve gNBs"}`,
		},
		{
			name:         "UpfEmptyDB",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "[]",
		},
		{
			name:         "OneUpf",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientOneUpf{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"hostname":"upf1","port":"123"}]`,
		},
		{
			name:         "ManyUpfs",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientManyUpfs{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"hostname":"upf0","port":"12"},{"hostname":"upf1","port":"345"},{"hostname":"upf2","port":"678"}]`,
		},
		{
			name:         "UpfDBError",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to retrieve UPFs"}`,
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

func TestGnbPostHandlers_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		inputData    string
		header       string
		expectedBody string
	}{
		{
			name:         "TAC is not a string",
			route:        "/config/v1/inventory/gnb/gnb1",
			inputData:    `{"tac": 1234}`,
			header:       "application/json",
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing TAC",
			route:        "/config/v1/inventory/gnb/gnb1",
			inputData:    `{"some_param": "123"}`,
			header:       "application/json",
			expectedBody: `{"error":"post gNB request body is missing tac"}`,
		},
		{
			name:         "GnbInvalidHeader",
			route:        "/config/v1/inventory/gnb/gnb1",
			inputData:    `{"tac": "123"}`,
			header:       "application",
			expectedBody: `{"error":"invalid header"}`,
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
			req.Header.Set("Content-Type", tc.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			expectedCode := http.StatusBadRequest
			if expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
			select {
			case msg := <-configChannel:
				t.Errorf("unexpected message received: %+v", msg)
			default:
				// This is the expected outcome (no message received)
			}
		})
	}
}

func TestGnbPostHandlers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name            string
		route           string
		inputData       string
		expectedMessage configmodels.ConfigMessage
	}{
		{
			name:      "PostGnb",
			route:     "/config/v1/inventory/gnb/gnb1",
			inputData: `{"tac": "123"}`,
			expectedMessage: configmodels.ConfigMessage{
				MsgType:   configmodels.Inventory,
				MsgMethod: configmodels.Post_op,
				Gnb: &configmodels.Gnb{
					Name: "gnb1",
					Tac:  "123",
				},
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

			expectedCode := http.StatusOK
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
				if tc.expectedMessage.Gnb != nil {
					if msg.Gnb == nil {
						t.Errorf("expected gNB %+v, but got nil", tc.expectedMessage.Gnb)
					}
					if tc.expectedMessage.Gnb.Name != msg.Gnb.Name || tc.expectedMessage.Gnb.Tac != msg.Gnb.Tac {
						t.Errorf("expected gNB %+v, but got %+v", tc.expectedMessage.Gnb, msg.Gnb)
					}
				}
			default:
				t.Error("expected message in configChannel, but none received")
			}
		})
	}
}

func TestUpfPostHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Create a new UPF success",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "host", "port": "123"}`,
			expectedCode: http.StatusCreated,
			expectedBody: "{}",
		},
		{
			name:         "Create an existing UPF expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientDuplicateCreation{},
			inputData:    `{"hostname": "upf1", "port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF already exists"}`,
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "host", "port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "host", "some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF port cannot be converted to integer or it was not provided"}`,
		},
		{
			name:         "DB POST operation fails expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"hostname": "host", "port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to create UPF"}`,
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "host", "port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF port cannot be converted to integer or it was not provided"}`,
		},
		{
			name:         "Hostname not provided expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF hostname must be provided"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(tc.inputData))
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

func TestUpfPutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Put a new UPF expects OK status",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Put an existing UPF expects a OK status",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientPutExistingUpf{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF port cannot be converted to integer or it was not provided"}`,
		},
		{
			name:         "DB PUT operation fails expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to PUT UPF"}`,
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF port cannot be converted to integer or it was not provided"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPut, tc.route, strings.NewReader(tc.inputData))
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

func TestInventoryDeleteHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Delete gNB Success",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Delete gNB DB Failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to delete gNB"}`,
		},
		{
			name:         "Delete UPF Success",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Delete UPF DB Failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to delete UPF"}`,
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
