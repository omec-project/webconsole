// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/dbadapter"
)

const (
	SUCCESS_BODY = `{"Result":"Operation Executed"}`
	BEARER       = "Bearer "
)

var (
	mockJWTSecret = []byte("mockSecret")
)

func MockOperation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Result": "Operation Executed"})
}

func setUpRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	dbadapter.WebuiDBClient = &MockMongoClientSuccess{}
	router.Use(AuthMiddleware(mockJWTSecret))
	AddService(router, mockJWTSecret)
	configapi.AddServiceSub(router)
	configapi.AddService(router)
	return router
}

func setUpMockedRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	dbadapter.WebuiDBClient = &MockMongoClientSuccess{}
	router.Use(AuthMiddleware(mockJWTSecret))
	router.GET("/config/v1/account", MockOperation)
	router.GET("/config/v1/account/:username", MockOperation)
	router.DELETE("/config/v1/account/:username", MockOperation)
	router.POST("/config/v1/account/:username/change_password", MockOperation)
	router.POST("/config/v1/account", MockOperation)
	return router
}

func TestMiddleware_NoHeaderRequest(t *testing.T) {
	router := setUpRouter()
	protectedPaths := []struct {
		name   string
		method string
		url    string
	}{
		{
			name:   "GetUserAccount",
			method: http.MethodGet,
			url:    "/config/v1/account/janedoe",
		},
		{
			name:   "GetUserAccounts",
			method: http.MethodGet,
			url:    "/config/v1/account",
		},
		{
			name:   "PostSecondUserAccount",
			method: http.MethodPost,
			url:    "/config/v1/account",
		},
		{
			name:   "DeleteUserAccount",
			method: http.MethodDelete,
			url:    "/config/v1/account/janedoe",
		},
		{
			name:   "ChangePassword",
			method: http.MethodPost,
			url:    "/config/v1/account/janedoe/change_password",
		},
		{
			name:   "ConfigV1",
			method: http.MethodGet,
			url:    "/config/v1/",
		},
		{
			name:   "GetDeviceGroups",
			method: http.MethodGet,
			url:    "/config/v1/device-group",
		},
		{
			name:   "GetDeviceGroupByName",
			method: http.MethodGet,
			url:    "/config/v1/device-group/some-name",
		},
		{
			name:   "DeviceGroupGroupNameDelete",
			method: http.MethodDelete,
			url:    "/config/v1/device-group/some-name",
		},
		{
			name:   "DeviceGroupGroupNamePatch",
			method: http.MethodPatch,
			url:    "/config/v1/device-group/some-name",
		},
		{
			name:   "DeviceGroupGroupNamePut",
			method: http.MethodPut,
			url:    "/config/v1/device-group/some-name",
		},
		{
			name:   "DeviceGroupGroupNamePost",
			method: http.MethodPost,
			url:    "/config/v1/device-group/some-name",
		},
		{
			name:   "GetNetworkSlices",
			method: http.MethodGet,
			url:    "/config/v1/network-slice",
		},
		{
			name:   "GetNetworkSliceByName",
			method: http.MethodGet,
			url:    "/config/v1/network-slice/some-slice",
		},
		{
			name:   "NetworkSliceSliceNameDelete",
			method: http.MethodDelete,
			url:    "/config/v1/network-slice/some-slice",
		},
		{
			name:   "NetworkSliceSliceNamePost",
			method: http.MethodPost,
			url:    "/config/v1/network-slice/some-slice",
		},
		{
			name:   "NetworkSliceSliceNamePut",
			method: http.MethodPut,
			url:    "/config/v1/network-slice/some-slice",
		},
		{
			name:   "GetGnbs",
			method: http.MethodGet,
			url:    "/config/v1/inventory/gnb",
		},
		{
			name:   "PostGnb",
			method: http.MethodPost,
			url:    "/config/v1/inventory/gnb/gnb-name",
		},
		{
			name:   "DeleteGnb",
			method: http.MethodDelete,
			url:    "/config/v1/inventory/gnb/gnb-name",
		},
		{
			name:   "GetUpfs",
			method: http.MethodGet,
			url:    "/config/v1/inventory/upf",
		},
		{
			name:   "PostUpf",
			method: http.MethodPost,
			url:    "/config/v1/inventory/upf/upf-name",
		},
		{
			name:   "DeleteUpf",
			method: http.MethodDelete,
			url:    "/config/v1/inventory/upf/upf-name",
		},
		{
			name:   "ApiSample",
			method: http.MethodGet,
			url:    "/api/sample",
		},
		{
			name:   "GetSubscribers",
			method: http.MethodGet,
			url:    "/api/subscriber",
		},
		{
			name:   "GetSubscriberByID",
			method: http.MethodGet,
			url:    "/api/subscriber/some-subs",
		},
		{
			name:   "PostSubscriberByID",
			method: http.MethodPost,
			url:    "/api/subscriber/some-subs",
		},
		{
			name:   "PutSubscriberByID",
			method: http.MethodPut,
			url:    "/api/subscriber/some-subs/plmnid",
		},
		{
			name:   "DeleteSubscriberByID",
			method: http.MethodDelete,
			url:    "/api/subscriber/some-subs",
		},
		{
			name:   "RegisteredUEContext",
			method: http.MethodGet,
			url:    "/api/registered-ue-context",
		},
		{
			name:   "IndividualRegisteredUEContext",
			method: http.MethodGet,
			url:    "/api/registered-ue-context/mysupi",
		},
		{
			name:   "UEPDUSessionInfo",
			method: http.MethodGet,
			url:    "/api/ue-pdu-session-info/smContextRef",
		},
	}

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			expectedBody := `{"error":"auth failed: authorization header not found"}`
			if expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != expectedBody {
				t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
			}
		})
	}
}

func TestPostUserAccount_CreateFirstUserWithoutHeader(t *testing.T) {
	router := setUpMockedRouter()
	dbadapter.WebuiDBClient = &MockMongoClientEmptyDB{}
	req, err := http.NewRequest(http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	expectedCode := http.StatusOK
	expectedBody := SUCCESS_BODY
	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if w.Body.String() != expectedBody {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}
}

func TestMiddleware_TokenValidation(t *testing.T) {
	router := setUpRouter()

	tests := []struct {
		name         string
		header       string
		expectedBody string
	}{
		{
			name:         "MissingToken",
			header:       "Bearer",
			expectedBody: `{"error":"auth failed: authorization header couldn't be processed. The expected format is 'Bearer token'"}`,
		},
		{
			name:         "InvalidToken",
			header:       "Bearer mytoken",
			expectedBody: `{"error":"auth failed: token is not valid"}`,
		},
		{
			name:         "MissingBearerKeyword",
			header:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Im5ld1VzZXIiLCJwZXJtaXNzaW9ucyI6MCwiZXhwIjoxNzI1OTYxOTUyfQ.r4U4RMaXZdDUYpL2tpNU1LNeN_Srzws0BzOW9coa7sg",
			expectedBody: `{"error":"auth failed: authorization header couldn't be processed. The expected format is 'Bearer token'"}`,
		},
		{
			name:         "MissingBearerAndToken",
			header:       "",
			expectedBody: `{"error":"auth failed: authorization header not found"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/config/v1/", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			if expectedCode != w.Code {
				t.Errorf("Expected status code `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("Expected body `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetUserAccounts_Authorization(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		permissions  int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "AdminUser_GetUserAccounts",
			username:     "janedoe",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "RegularUser_GetUserAccounts",
			username:     "someuser",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := generateJWT(tc.username, tc.permissions, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := BEARER + jwtToken
			req.Header.Set("Authorization", validToken)
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

func TestGetUserAccount_Authorization(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		permissions  int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_GetOwnUserAccount",
			username:     "janedoe",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "AdminUser_GetOwnUserAccount",
			username:     "janedoe",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "RegularUser_GetOtherUserAccount",
			username:     "someuser",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
		{
			name:         "AdminUser_GetOtherUserAccount",
			username:     "someuser",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account/janedoe", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := generateJWT(tc.username, tc.permissions, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := BEARER + jwtToken
			req.Header.Set("Authorization", validToken)
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

func TestPostUserAccount_Authorization(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		permissions  int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "AdminUser_GetUserAccounts",
			username:     "janedoe",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "RegularUser_GetUserAccounts",
			username:     "someuser",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := generateJWT(tc.username, tc.permissions, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := BEARER + jwtToken
			req.Header.Set("Authorization", validToken)
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

func TestDeleteUserAccount_Authorization(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		permissions  int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_DeleteOwnUserAccount",
			username:     "janedoe",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
		{
			name:         "AdminUser_DeleteOwnUserAccount",
			username:     "janedoe",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "RegularUser_DeleteOtherUserAccount",
			username:     "someuser",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
		{
			name:         "AdminUser_DeleteOtherUserAccount",
			username:     "someuser",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, "/config/v1/account/janedoe", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := generateJWT(tc.username, tc.permissions, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := BEARER + jwtToken
			req.Header.Set("Authorization", validToken)
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

func TestChangePassword_Authorization(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		permissions  int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_OwnUserAccount",
			username:     "janedoe",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "AdminUser_OwnUserAccount",
			username:     "janedoe",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
		{
			name:         "RegularUser_OtherUserAccount",
			username:     "someuser",
			permissions:  USER_ACCOUNT,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden"}`,
		},
		{
			name:         "AdminUser_OtherUserAccount",
			username:     "someuser",
			permissions:  ADMIN_ACCOUNT,
			expectedCode: http.StatusOK,
			expectedBody: SUCCESS_BODY,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account/janedoe/change_password", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := generateJWT(tc.username, tc.permissions, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := BEARER + jwtToken
			req.Header.Set("Authorization", validToken)
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
