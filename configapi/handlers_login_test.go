// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
)

func TestLogin_FailureCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddAuthenticationService(router, mockJWTSecret)

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
	AddAuthenticationService(router, mockJWTSecret)

	testCases := []struct {
		name             string
		dbAdapter        dbadapter.DBInterface
		inputData        string
		expectedCode     int
		expectedUsername string
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
				} else if int(claims["role"].(float64)) != tc.expectedRole {
					t.Errorf("Expected `%v` role, got `%v`", tc.expectedRole, claims["role"])
				}
			} else {
				t.Errorf("Invalid JWT token or JWT claims are not readable")
			}
		})
	}
}
