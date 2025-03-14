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

func (db *MockMongoClientPutExistingUpf) RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error {
	return nil
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

func TestGnbPostHandler(t *testing.T) {
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
			name:         "Create a new gNB expects created status",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"name": "gnb1", "tac": "123"}`,
			expectedCode: http.StatusCreated,
			expectedBody: "{}",
		},
		{
			name:         "Create a new gNB without TAC expects created status",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"name": "gnb1"}`,
			expectedCode: http.StatusCreated,
			expectedBody: "{}",
		},
		{
			name:         "Create an existing gNB expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientDuplicateCreation{},
			inputData:    `{"name": "gnb1", "tac": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"gNB already exists"}`,
		},
		{
			name:         "TAC is not a string expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"name": "gnb1", "tac": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "DB POST operation fails expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"name": "gnb1", "tac": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to create gNB"}`,
		},
		{
			name:         "TAC cannot be converted to int expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"name": "gnb1", "tac": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB TAC 'a'. TAC must be a numeric string within the range [1, 16777215]"}`,
		},
		{
			name:         "gNB name not provided expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"tac": "12"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB name ''. Name needs to match the following regular expression: ^[a-zA-Z][a-zA-Z0-9-_]+$"}`,
		},
		{
			name:         "Invalid gNB name expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"name": "gn!b1", "tac": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB name 'gn!b1'. Name needs to match the following regular expression: ^[a-zA-Z][a-zA-Z0-9-_]+$"}`,
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

func TestGnbPutHandler(t *testing.T) {
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
			name:         "Put a new gNB expects OK status",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"tac": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Put an existing gNB expects a OK status",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientPutExistingUpf{},
			inputData:    `{"tac": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "TAC is not a string expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"tac": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing TAC expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB TAC ''. TAC must be a numeric string within the range [1, 16777215]"}`,
		},
		{
			name:         "DB PUT operation fails expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"tac": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to PUT gNB"}`,
		},
		{
			name:         "TAC cannot be converted to int expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"tac": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB TAC 'a'. TAC must be a numeric string within the range [1, 16777215]"}`,
		},
		{
			name:         "Invalid gNB name expects failure",
			route:        "/config/v1/inventory/gnb/gn!b1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"tac": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid gNB name 'gn!b1'. Name needs to match the following regular expression: ^[a-zA-Z][a-zA-Z0-9-_]+$"}`,
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
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusCreated,
			expectedBody: "{}",
		},
		{
			name:         "Create an existing UPF expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientDuplicateCreation{},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"UPF already exists"}`,
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "upf1.my-domain.com", "some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF port ''. Port must be a numeric string within the range [0, 65535]"}`,
		},
		{
			name:         "DB POST operation fails expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to create UPF"}`,
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF port 'a'. Port must be a numeric string within the range [0, 65535]"}`,
		},
		{
			name:         "Hostname not provided expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF hostname ''. Hostname needs to represent a valid FQDN"}`,
		},
		{
			name:         "Invalid UPF hostname expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"hostname": "upf1", "port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF hostname 'upf1'. Hostname needs to represent a valid FQDN"}`,
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
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Put an existing UPF expects a OK status",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientPutExistingUpf{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid JSON format"}`,
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF port ''. Port must be a numeric string within the range [0, 65535]"}`,
		},
		{
			name:         "DB PUT operation fails expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"failed to PUT UPF"}`,
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF port 'a'. Port must be a numeric string within the range [0, 65535]"}`,
		},
		{
			name:         "Invalid UPF hostname expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"invalid UPF hostname 'upf1'. Hostname needs to represent a valid FQDN"}`,
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
