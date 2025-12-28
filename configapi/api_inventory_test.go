// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"errors"
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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func gnb(name string, tac int32) configmodels.Gnb {
	return configmodels.Gnb{
		Name: name,
		Tac:  &tac,
	}
}

type GnbMockDBClient struct {
	dbadapter.DBInterface
	gnbs []configmodels.Gnb
	err  error
}

func (db *GnbMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if coll == sliceDataColl {
		return nil, nil
	}
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, g := range db.gnbs {
		gnb := configmodels.ToBsonM(g)
		if gnb == nil {
			logger.AppLog.Fatalln("failed to convert gnbs to BsonM")
		}
		results = append(results, gnb)
	}
	return results, db.err
}

func (db *GnbMockDBClient) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	if db.err != nil {
		return false, db.err
	}
	if len(db.gnbs) == 0 {
		return false, nil
	}
	return true, nil // Return true if data exists
}

func (db *GnbMockDBClient) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

func (db *GnbMockDBClient) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	if db.err != nil {
		return db.err
	}
	if len(db.gnbs) == 0 {
		return nil
	}
	return errors.New("E11000")
}

func (db *GnbMockDBClient) RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	return nil
}

func upf(hostname, port string) configmodels.Upf {
	return configmodels.Upf{
		Hostname: hostname,
		Port:     port,
	}
}

type UpfMockDBClient struct {
	dbadapter.DBInterface
	upfs []configmodels.Upf
	err  error
}

func (db *UpfMockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	if coll == sliceDataColl {
		return nil, nil
	}
	if db.err != nil {
		return nil, db.err
	}
	var results []map[string]any
	for _, u := range db.upfs {
		upf := configmodels.ToBsonM(u)
		if upf == nil {
			logger.AppLog.Fatalln("failed to convert upfs to BsonM")
		}
		results = append(results, upf)
	}
	return results, db.err
}

func (db *UpfMockDBClient) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	if db.err != nil {
		return false, db.err
	}
	if len(db.upfs) == 0 {
		return false, nil
	}
	return true, nil // Return true if data exists
}

func (db *UpfMockDBClient) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

func (db *UpfMockDBClient) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	if db.err != nil {
		return db.err
	}
	if len(db.upfs) == 0 {
		return nil
	}
	return errors.New("E11000")
}

func (db *UpfMockDBClient) RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error {
	if db.err != nil {
		return db.err
	}
	return nil
}

func TestInventoryGetGnbHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name           string
		route          string
		configuredGnbs []configmodels.Gnb
		expectedCode   int
		expectedBody   []configmodels.Gnb
	}{
		{
			name:           "GnbEmptyDB",
			route:          "/config/v1/inventory/gnb",
			configuredGnbs: []configmodels.Gnb{},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Gnb{},
		},
		{
			name:           "OneGnb",
			route:          "/config/v1/inventory/gnb",
			configuredGnbs: []configmodels.Gnb{gnb("gnb1", 123)},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Gnb{gnb("gnb1", 123)},
		},
		{
			name:           "ManyGnbs",
			route:          "/config/v1/inventory/gnb",
			configuredGnbs: []configmodels.Gnb{gnb("gnb0", 12), gnb("gnb1", 345), gnb("gnb2", 678)},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Gnb{gnb("gnb0", 12), gnb("gnb1", 345), gnb("gnb2", 678)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()

			dbadapter.CommonDBClient = &GnbMockDBClient{
				gnbs: tc.configuredGnbs,
			}
			req, err := http.NewRequest(http.MethodGet, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual []configmodels.Gnb
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected %+v, got %+v", expected, actual)
			}
		})
	}
}

func TestInventoryGetUPFHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name           string
		route          string
		configuredUpfs []configmodels.Upf
		expectedCode   int
		expectedBody   []configmodels.Upf
	}{
		{
			name:           "UpfEmptyDB",
			route:          "/config/v1/inventory/upf",
			configuredUpfs: []configmodels.Upf{},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Upf{},
		},
		{
			name:           "OneUpf",
			route:          "/config/v1/inventory/upf",
			configuredUpfs: []configmodels.Upf{upf("upf1", "123")},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Upf{upf("upf1", "123")},
		},
		{
			name:           "ManyUpfs",
			route:          "/config/v1/inventory/upf",
			configuredUpfs: []configmodels.Upf{upf("upf0", "12"), upf("upf1", "345"), upf("upf2", "678")},
			expectedCode:   http.StatusOK,
			expectedBody:   []configmodels.Upf{upf("upf0", "12"), upf("upf1", "345"), upf("upf2", "678")},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()

			dbadapter.CommonDBClient = &UpfMockDBClient{
				upfs: tc.configuredUpfs,
			}
			req, err := http.NewRequest(http.MethodGet, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual []configmodels.Upf
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected %+v, got %+v", expected, actual)
			}
		})
	}
}

func TestGetInventory_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody map[string]string
	}{
		{
			name:         "GnbDBError",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{err: fmt.Errorf("mock error")},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to retrieve gNBs"},
		},
		{
			name:         "UpfDBError",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{err: fmt.Errorf("mock error")},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to retrieve UPFs"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()

			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodGet, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
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
		expectedBody map[string]string
	}{
		{
			name:         "Create a new gNB expects created status",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"name": "gnb1", "tac": 123}`,
			expectedCode: http.StatusCreated,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Create a new gNB without TAC expects created status",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"name": "gnb1"}`,
			expectedCode: http.StatusCreated,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Create an existing gNB expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{{Name: "gnb1"}}},
			inputData:    `{"name": "gnb1", "tac": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "gNB already exists"},
		},
		{
			name:         "TAC is not an integer expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"name": "gnb1", "tac": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid JSON format"},
		},
		{
			name:         "TAC is zero expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"name": "gnb1", "tac": 0}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB TAC '0'. TAC must be an integer within the range [1, 16777215]"},
		},
		{
			name:         "DB POST operation fails expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{err: fmt.Errorf("mock error")},
			inputData:    `{"name": "gnb1", "tac": 123}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to create gNB"},
		},
		{
			name:         "gNB name not provided expects failure",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"tac": 12}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB name ''. Name needs to match the following regular expression: " + NAME_PATTERN},
		},
		{
			name:         "Invalid gNB name expects failure (invalid token)",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"name": "gn!b1", "tac": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB name 'gn!b1'. Name needs to match the following regular expression: " + NAME_PATTERN},
		},
		{
			name:         "Invalid gNB name expects failure (invalid length)",
			route:        "/config/v1/inventory/gnb",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    "{\"name\": \"" + genLongString(257) + "\", \"tac\": 123}",
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB name '" + genLongString(257) + "'. Name needs to match the following regular expression: " + NAME_PATTERN},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
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
		expectedBody map[string]string
	}{
		{
			name:         "Put a new gNB expects OK status",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"tac": 123}`,
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Put an existing gNB expects a OK status",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{{Name: "name"}}},
			inputData:    `{"tac": 123}`,
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "TAC is not an integer expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"tac": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid JSON format"},
		},
		{
			name:         "Missing TAC expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"some_param": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB TAC '0'. TAC must be an integer within the range [1, 16777215]"},
		},
		{
			name:         "DB PUT operation fails expects failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{err: fmt.Errorf("mock error")},
			inputData:    `{"tac": 123}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to PUT gNB"},
		},
		{
			name:         "Invalid gNB name expects failure",
			route:        "/config/v1/inventory/gnb/gn!b1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			inputData:    `{"tac": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid gNB name 'gn!b1'. Name needs to match the following regular expression: " + NAME_PATTERN},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPut, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
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
		expectedBody map[string]string
	}{
		{
			name:         "Create a new UPF success",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusCreated,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Create an existing UPF expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{upf("upf1.my-domain.com", "123")}},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "UPF already exists"},
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid JSON format"},
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"hostname": "upf1.my-domain.com", "some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF port ''. Port must be a numeric string within the range [0, 65535]"},
		},
		{
			name:         "DB POST operation fails expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{err: fmt.Errorf("mock error")},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to create UPF"},
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"hostname": "upf1.my-domain.com", "port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF port 'a'. Port must be a numeric string within the range [0, 65535]"},
		},
		{
			name:         "Hostname not provided expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF hostname ''. Hostname needs to represent a valid FQDN"},
		},
		{
			name:         "Invalid UPF hostname expects failure",
			route:        "/config/v1/inventory/upf",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"hostname": "upf1", "port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF hostname 'upf1'. Hostname needs to represent a valid FQDN"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
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
		expectedBody map[string]string
	}{
		{
			name:         "Put a new UPF expects OK status",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Put an existing UPF expects a OK status",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{upf("upf1.my-domain.com", "123")}},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Port is not a string expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"port": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid JSON format"},
		},
		{
			name:         "Missing port expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"some_param": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF port ''. Port must be a numeric string within the range [0, 65535]"},
		},
		{
			name:         "DB PUT operation fails expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{err: fmt.Errorf("mock error")},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to PUT UPF"},
		},
		{
			name:         "Port cannot be converted to int expects failure",
			route:        "/config/v1/inventory/upf/upf1.my-domain.com",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"port": "a"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF port 'a'. Port must be a numeric string within the range [0, 65535]"},
		},
		{
			name:         "Invalid UPF hostname expects failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			inputData:    `{"port": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: map[string]string{"error": "invalid UPF hostname 'upf1'. Hostname needs to represent a valid FQDN"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPut, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
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
		expectedBody map[string]string
	}{
		{
			name:         "Delete gNB Success",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{gnbs: []configmodels.Gnb{}},
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Delete gNB DB Failure",
			route:        "/config/v1/inventory/gnb/gnb1",
			dbAdapter:    &GnbMockDBClient{err: fmt.Errorf("mock error")},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to delete gNB"},
		},
		{
			name:         "Delete UPF Success",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &UpfMockDBClient{upfs: []configmodels.Upf{}},
			expectedCode: http.StatusOK,
			expectedBody: make(map[string]string),
		},
		{
			name:         "Delete UPF DB Failure",
			route:        "/config/v1/inventory/upf/upf1",
			dbAdapter:    &UpfMockDBClient{err: fmt.Errorf("mock error")},
			expectedCode: http.StatusInternalServerError,
			expectedBody: map[string]string{"error": "failed to delete UPF"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodDelete, tc.route, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			bodyBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			var actual map[string]string
			if err := json.Unmarshal(bodyBytes, &actual); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}
			expected := tc.expectedBody
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("expected response body %v, got %v", expected, actual)
			}
		})
	}
}
