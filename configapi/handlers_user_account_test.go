// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

const (
	testChangePasswordBody = `{"password": "Admin1234"}`
	testCaseDBError        = "DBError"
)

const (
	testUserJohndoe = "johndoe"
	testUserJanedoe = "janedoe"
	passwordKey     = "password"
	roleKey         = "role"
)

type MockMongoClientInvalidUser struct {
	dbadapter.DBInterface
}

type MockMongoClientSuccess struct {
	dbadapter.DBInterface
}

type MockMongoClientRegularUser struct {
	dbadapter.DBInterface
}

func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hashed)
}

func (db *MockMongoClientInvalidUser) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	rawUser := map[string]any{
		usernameKey: testUserJohndoe,
		passwordKey: 1234,
		roleKey:     "a",
	}
	return rawUser, nil
}

func (db *MockMongoClientInvalidUser) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	rawUsers := []map[string]any{
		{usernameKey: testUserJohndoe, passwordKey: 1234, roleKey: "a"},
		{usernameKey: testUserJanedoe, passwordKey: hashPassword("Password123"), roleKey: 1},
	}
	return rawUsers, nil
}

func (db *MockMongoClientSuccess) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	rawUser := map[string]any{
		usernameKey: testUserJanedoe, passwordKey: hashPassword("password123!"), roleKey: 1,
	}
	return rawUser, nil
}

func (db *MockMongoClientSuccess) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	rawUsers := []map[string]any{
		{usernameKey: testUserJohndoe, passwordKey: hashPassword(".password123"), roleKey: 0},
		{usernameKey: testUserJanedoe, passwordKey: hashPassword("password123"), roleKey: 1},
	}
	return rawUsers, nil
}

func (db *MockMongoClientSuccess) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return true, nil
}

func (db *MockMongoClientSuccess) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 5, nil
}

func (db *MockMongoClientRegularUser) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	rawUser := map[string]any{
		usernameKey: testUserJohndoe, passwordKey: hashPassword("password-123"), roleKey: 0,
	}
	return rawUser, nil
}

func (db *MockMongoClientRegularUser) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return nil
}

type MockMongoClientDuplicateCreation struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	return results, nil
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	return errors.New("E11000")
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	return errors.New("E11000")
}

func (db *MockMongoClientDuplicateCreation) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 1, nil
}

func (m *MockMongoClientDuplicateCreation) StartSession() (dbadapter.DBSession, error) {
	return nil, nil
}

func TestGetUserAccountsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/config/v1/account", GetUserAccounts)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         testCaseDBError,
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccounts),
		},
		{
			name:         "DBReturnsOneInvalidUser",
			dbAdapter:    &MockMongoClientInvalidUser{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"username":"janedoe","role":1}]`,
		},
		{
			name:         "NoUsersInDB",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: "[]",
		},
		{
			name:         "SuccessManyUsers",
			dbAdapter:    &MockMongoClientSuccess{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"username":"johndoe","role":0},{"username":"janedoe","role":1}]`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/config/v1/account", nil)
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

func TestGetUserAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/config/v1/account/:username", GetUserAccount)

	testCases := []struct {
		name         string
		username     string
		role         int
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "GetUserAccountSuccess",
			dbAdapter:    &MockMongoClientSuccess{},
			expectedCode: http.StatusOK,
			expectedBody: `{"username":"janedoe","role":1}`,
		},
		{
			name:         testCaseDBError,
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "UserNotFound",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusNotFound,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorUsernameNotFound),
		},
		{
			name:         "InvalidUser",
			dbAdapter:    &MockMongoClientInvalidUser{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/config/v1/account/janedoe", nil)
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

func TestCreateUserAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/config/v1/account", CreateUserAccount)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RequestWithoutUsername",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"password" : "Admin1234"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorMissingUsername),
		},
		{
			name:         "UserThatAlreadyExists",
			dbAdapter:    &MockMongoClientDuplicateCreation{},
			inputData:    `{"username": "janedoe", "password" : "Admin1234"}`,
			expectedCode: http.StatusConflict,
			expectedBody: `{"error":"user account already exists"}`,
		},
		{
			name:         "RequestWithoutPassword",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"username": "adminadmin"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorMissingPassword),
		},
		{
			name:         "SuccessfulRequest",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"username": "adminadmin", "password" : "Admin1234"}`,
			expectedCode: http.StatusCreated,
			expectedBody: `{}`,
		},
		{
			name:         testCaseDBError,
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"username": "adminadmin", "password" : "Admin1234"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccounts),
		},
		{
			name:         "InvalidPassword",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"username": "adminadmin", "password" : "1234"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorInvalidPassword),
		},
		{
			name:         "InvalidJsonProvided",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"username": "adminadmin", "password": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorInvalidDataProvided),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/account", strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
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

func TestDeleteUserAccountHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.DELETE("/config/v1/account/:username", DeleteUserAccount)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Success",
			dbAdapter:    &MockMongoClientRegularUser{},
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "DeleteAdminUser",
			dbAdapter:    &MockMongoClientSuccess{},
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorDeleteAdminAccount),
		},
		{
			name:         "DeleteInvalidUser",
			dbAdapter:    &MockMongoClientInvalidUser{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "UserNotFound",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusNotFound,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorUsernameNotFound),
		},
		{
			name:         testCaseDBError,
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/config/v1/account/janedoe", nil)
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

func TestChangePasswordHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/config/v1/account/:username/change_password", ChangeUserAccountPasssword)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Success",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    testChangePasswordBody,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         testCaseDBError,
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    testChangePasswordBody,
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "UserDoesNotExist",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    testChangePasswordBody,
			expectedCode: http.StatusNotFound,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorUsernameNotFound),
		},
		{
			name:         "InvalidPassword",
			dbAdapter:    nil,
			inputData:    `{"password": "1234"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorInvalidPassword),
		},
		{
			name:         "NoPasswordProvided",
			dbAdapter:    nil,
			inputData:    `{}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorMissingPassword),
		},
		{
			name:         "InvalidData",
			dbAdapter:    nil,
			inputData:    `{"password": 1234}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorInvalidDataProvided),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/account/janedoe/change_password", strings.NewReader(tc.inputData))
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
