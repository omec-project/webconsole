// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type MockMongoClientEmptyDB struct {
	dbadapter.DBInterface
}

type MockMongoClientDBError struct {
	dbadapter.DBInterface
}

type MockMongoClientInvalidUser struct {
	dbadapter.DBInterface
}

type MockMongoClientSuccess struct {
	dbadapter.DBInterface
}

type MockMongoClientRegularUser struct {
	dbadapter.DBInterface
}

type MockMongoClientDuplicateCreation struct {
	dbadapter.DBInterface
}

func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hashed)
}

func (m *MockMongoClientEmptyDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockMongoClientEmptyDB) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (m *MockMongoClientEmptyDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return nil
}

func (m *MockMongoClientDBError) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return nil, errors.New("DB error")
}

func (m *MockMongoClientDBError) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	return nil, errors.New("DB error")
}

func (m *MockMongoClientDBError) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return false, errors.New("DB error")
}

func (m *MockMongoClientInvalidUser) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	rawUser := map[string]interface{}{
		"username":    "johndoe",
		"password":    1234,
		"permissions": 0,
	}
	return rawUser, nil
}

func (m *MockMongoClientInvalidUser) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	rawUsers := []map[string]interface{}{
		{"username": "johndoe", "password": 1234, "permissions": 0},
		{"username": "janedoe", "password": hashPassword("Password123"), "permissions": 1},
	}
	return rawUsers, nil
}

func (m *MockMongoClientSuccess) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	rawUser := map[string]interface{}{
		"username": "janedoe", "password": hashPassword("password123!"), "permissions": 1,
	}
	return rawUser, nil
}

func (m *MockMongoClientSuccess) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	rawUsers := []map[string]interface{}{
		{"username": "johndoe", "password": hashPassword(".password123"), "permissions": 0},
		{"username": "janedoe", "password": hashPassword("password123"), "permissions": 1},
	}
	return rawUsers, nil
}

func (m *MockMongoClientSuccess) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return true, nil
}

func (m *MockMongoClientRegularUser) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	rawUser := map[string]interface{}{
		"username": "johndoe", "password": hashPassword("password-123"), "permissions": 0,
	}
	return rawUser, nil
}

func (m *MockMongoClientRegularUser) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return nil
}

func (m *MockMongoClientDuplicateCreation) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return errors.New("E11000")
}

func mockGeneratePassword() (string, error) {
	return "ValidPass123!", nil
}

func mockGeneratePasswordFailure() (string, error) {
	return "", errors.New("password generation failed")
}

func TestGetUserAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddService(router, mockJWTSecret)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "DBError",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccounts),
		},
		{
			name:         "DBReturnsOneInvalidUser",
			dbAdapter:    &MockMongoClientInvalidUser{},
			expectedCode: http.StatusOK,
			expectedBody: `[{"username":"janedoe","permissions":1}]`,
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
			expectedBody: `[{"username":"johndoe","permissions":0},{"username":"janedoe","permissions":1}]`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account", nil)
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

func TestGetUserAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddService(router, mockJWTSecret)

	testCases := []struct {
		name         string
		username     string
		permissions  int
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "GetUserAccountSuccess",
			dbAdapter:    &MockMongoClientSuccess{},
			expectedCode: http.StatusOK,
			expectedBody: `{"username":"janedoe","permissions":1}`,
		},
		{
			name:         "DBError",
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
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account/janedoe", nil)
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

func TestPostUserAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddService(router, mockJWTSecret)

	testCases := []struct {
		name                 string
		dbAdapter            dbadapter.DBInterface
		generatePasswordMock func() (string, error)
		inputData            string
		expectedCode         int
		expectedBody         string
	}{
		{
			name:                 "RequestWithoutUsername",
			dbAdapter:            &MockMongoClientSuccess{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            "{}",
			expectedCode:         http.StatusBadRequest,
			expectedBody:         fmt.Sprintf(`{"error":"%s"}`, errorMissingUsername),
		},
		{
			name:                 "UserThatAlreadyExists",
			dbAdapter:            &MockMongoClientDuplicateCreation{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "janedoe"}`,
			expectedCode:         http.StatusBadRequest,
			expectedBody:         `{"error":"user account already exists"}`,
		},
		{
			name:                 "RequestWithoutPassword",
			dbAdapter:            &MockMongoClientEmptyDB{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "adminadmin"}`,
			expectedCode:         http.StatusCreated,
			expectedBody:         `{"password":"ValidPass123!"}`,
		},
		{
			name:                 "RequestWithPassword",
			dbAdapter:            &MockMongoClientEmptyDB{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "adminadmin", "password" : "Admin1234"}`,
			expectedCode:         http.StatusCreated,
			expectedBody:         `{}`,
		},
		{
			name:                 "DBError",
			dbAdapter:            &MockMongoClientDBError{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "adminadmin", "password" : "Admin1234"}`,
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccounts),
		},
		{
			name:                 "InvalidPassword",
			dbAdapter:            &MockMongoClientSuccess{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "adminadmin", "password" : "1234"}`,
			expectedCode:         http.StatusBadRequest,
			expectedBody:         fmt.Sprintf(`{"error":"%s"}`, errorInvalidPassword),
		},
		{
			name:                 "ErrorGeneratingPassword",
			dbAdapter:            &MockMongoClientSuccess{},
			generatePasswordMock: mockGeneratePasswordFailure,
			inputData:            `{"username": "adminadmin"}`,
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         fmt.Sprintf(`{"error":"%s"}`, errorCreateUserAccount),
		},
		{
			name:                 "InvalidJsonProvided",
			dbAdapter:            &MockMongoClientSuccess{},
			generatePasswordMock: mockGeneratePassword,
			inputData:            `{"username": "adminadmin", "password": 1234}`,
			expectedCode:         http.StatusBadRequest,
			expectedBody:         fmt.Sprintf(`{"error":"%s"}`, errorInvalidDataProvided),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generatePassword = tc.generatePasswordMock
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account", strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
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

func TestDeleteUserAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddService(router, mockJWTSecret)

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
			name:         "DBError",
			dbAdapter:    &MockMongoClientDBError{},
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodDelete, "/config/v1/account/janedoe", nil)
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

func TestChangePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddService(router, mockJWTSecret)

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
			inputData:    `{"password": "Admin1234"}`,
			expectedCode: http.StatusOK,
			expectedBody: "{}",
		},
		{
			name:         "DBError",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"password": "Admin1234"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "UserDoesNotExist",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"password": "Admin1234"}`,
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
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account/janedoe/change_password", strings.NewReader(tc.inputData))
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

func TestLogin_FailureCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	router.Use(AuthMiddleware(mockJWTSecret))
	AddService(router, mockJWTSecret)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "InvalidDataProvided",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"username":"testuser", "password": 123}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorInvalidDataProvided),
		},
		{
			name:         "NoUsernameProvided",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"password": "123"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorMissingUsername),
		},
		{
			name:         "NoPasswordProvided",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"username":"testuser"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorMissingPassword),
		},
		{
			name:         "DBError",
			dbAdapter:    &MockMongoClientDBError{},
			inputData:    `{"username":"testuser", "password":"password123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "UserNotFound",
			dbAdapter:    &MockMongoClientEmptyDB{},
			inputData:    `{"username":"testuser", "password":"password123"}`,
			expectedCode: http.StatusUnauthorized,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorIncorrectCredentials),
		},
		{
			name:         "InvalidUserObtainedFromDB",
			dbAdapter:    &MockMongoClientInvalidUser{},
			inputData:    `{"username":"testuser", "password":"password123"}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorRetrieveUserAccount),
		},
		{
			name:         "IncorrectPassword",
			dbAdapter:    &MockMongoClientSuccess{},
			inputData:    `{"username":"testuser", "password":"a-password"}`,
			expectedCode: http.StatusUnauthorized,
			expectedBody: fmt.Sprintf(`{"error":"%s"}`, errorIncorrectCredentials),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.inputData))
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

func TestLogin_SuccessCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	router.Use(AuthMiddleware(mockJWTSecret))
	AddService(router, mockJWTSecret)

	testCases := []struct {
		name                string
		dbAdapter           dbadapter.DBInterface
		inputData           string
		expectedCode        int
		expectedUsername    string
		expectedPermissions int
	}{
		{
			name:                "Success_AdminUser",
			dbAdapter:           &MockMongoClientSuccess{},
			inputData:           `{"username":"janedoe", "password":"password123!"}`,
			expectedCode:        http.StatusOK,
			expectedUsername:    "janedoe",
			expectedPermissions: ADMIN_ACCOUNT,
		},
		{
			name:                "Success_RegularUser",
			dbAdapter:           &MockMongoClientRegularUser{},
			inputData:           `{"username":"johndoe", "password":"password-123"}`,
			expectedCode:        http.StatusOK,
			expectedUsername:    "johndoe",
			expectedPermissions: USER_ACCOUNT,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			var responde_data map[string]string
			err = json.Unmarshal([]byte(w.Body.Bytes()), &responde_data)
			if err != nil {
				t.Errorf("Unable to unmarshal response`%v`", w.Body.String())
			}

			response_token, exists := responde_data["token"]
			if !exists {
				t.Errorf("Unable to unmarshal response`%v`", w.Body.String())
			}

			token, parseErr := jwt.Parse(response_token, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(mockJWTSecret), nil
			})
			if parseErr != nil {
				t.Errorf("Error parsing JWT: %v", parseErr)
				return
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				if claims["username"] != tc.expectedUsername {
					t.Errorf("Expected `%v` username, got `%v`", tc.expectedUsername, claims["username"])
				} else if int(claims["permissions"].(float64)) != tc.expectedPermissions {
					t.Errorf("Expected `%v` permissions, got `%v`", tc.expectedPermissions, claims["permissions"])
				}
			} else {
				t.Errorf("Invalid JWT token or JWT claims are not readable")
			}
		})
	}
}
