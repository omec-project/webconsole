// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

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
	"github.com/golang-jwt/jwt/v5"
	"github.com/omec-project/webconsole/configmodels"
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

func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hashed)
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	return map[string]any{}, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	return results, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, nil
}

func (db *MockMongoClientDBError) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, errors.New("DB error")
}

func (db *MockMongoClientInvalidUser) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	rawUser := map[string]any{
		"username": "johndoe",
		"password": 1234,
		"role":     "a",
	}
	return rawUser, nil
}

func (db *MockMongoClientInvalidUser) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	rawUsers := []map[string]any{
		{"username": "johndoe", "password": 1234, "role": "a"},
		{"username": "janedoe", "password": hashPassword("Password123"), "role": 1},
	}
	return rawUsers, nil
}

func (db *MockMongoClientSuccess) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	rawUser := map[string]any{
		"username": "janedoe", "password": hashPassword("password123!"), "role": 1,
	}
	return rawUser, nil
}

func (db *MockMongoClientSuccess) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	rawUsers := []map[string]any{
		{"username": "johndoe", "password": hashPassword(".password123"), "role": 0},
		{"username": "janedoe", "password": hashPassword("password123"), "role": 1},
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
		"username": "johndoe", "password": hashPassword("password-123"), "role": 0,
	}
	return rawUser, nil
}

func (db *MockMongoClientRegularUser) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return nil
}

func TestLogin_FailureCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddAuthenticationService(router, mockJWTSecret)

	testCases := []struct {
		dbAdapter    dbadapter.DBInterface
		name         string
		inputData    string
		expectedBody string
		expectedCode int
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestLogin_SuccessCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddAuthenticationService(router, mockJWTSecret)

	testCases := []struct {
		dbAdapter        dbadapter.DBInterface
		name             string
		inputData        string
		expectedUsername string
		expectedCode     int
		expectedRole     int
	}{
		{
			name:             "Success_AdminUser",
			dbAdapter:        &MockMongoClientSuccess{},
			inputData:        `{"username":"janedoe", "password":"password123!"}`,
			expectedCode:     http.StatusOK,
			expectedUsername: "janedoe",
			expectedRole:     configmodels.AdminRole,
		},
		{
			name:             "Success_RegularUser",
			dbAdapter:        &MockMongoClientRegularUser{},
			inputData:        `{"username":"johndoe", "password":"password-123"}`,
			expectedCode:     http.StatusOK,
			expectedUsername: "johndoe",
			expectedRole:     configmodels.UserRole,
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			var respondeData map[string]string
			err = json.Unmarshal(w.Body.Bytes(), &respondeData)
			if err != nil {
				t.Errorf("unable to unmarshal response`%v`", w.Body.String())
			}

			responseToken, exists := respondeData["token"]
			if !exists {
				t.Errorf("unable to unmarshal response`%v`", w.Body.String())
			}

			token, parseErr := jwt.Parse(responseToken, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return mockJWTSecret, nil
			})
			if parseErr != nil {
				t.Errorf("error parsing JWT: %v", parseErr)
				return
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				if claims["username"] != tc.expectedUsername {
					t.Errorf("expected `%v` username, got `%v`", tc.expectedUsername, claims["username"])
				} else if int(claims["role"].(float64)) != tc.expectedRole {
					t.Errorf("expected `%v` role, got `%v`", tc.expectedRole, claims["role"])
				}
			} else {
				t.Errorf("invalid JWT token or JWT claims are not readable")
			}
		})
	}
}
