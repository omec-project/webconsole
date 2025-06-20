// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/auth"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
)

const (
	successBody = `{"Result":"Operation Executed"}`
	bearer      = "Bearer "
)

var mockJWTSecret = []byte("mockSecret")

func MockOperation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Result": "Operation Executed"})
}

func setUpRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	dbadapter.WebuiDBClient = &MockMongoClientSuccess{}
	router.Use(auth.AdminOrUserAuthMiddleware(mockJWTSecret))
	AddUserAccountService(router, mockJWTSecret)
	AddApiService(router)
	AddConfigV1Service(router)
	return router
}

func setUpMockedRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	dbadapter.WebuiDBClient = &MockMongoClientSuccess{}
	router.GET("/config/v1/account", auth.AdminOnly(mockJWTSecret, MockOperation))
	router.GET("/config/v1/account/:username", auth.AdminOrMe(mockJWTSecret, MockOperation))
	router.DELETE("/config/v1/account/:username", auth.AdminOnly(mockJWTSecret, MockOperation))
	router.POST("/config/v1/account/:username/change_password", auth.AdminOrMe(mockJWTSecret, MockOperation))
	router.POST("/config/v1/account", auth.AdminOrFirstUser(mockJWTSecret, MockOperation))
	return router
}

func TestAdminOrUserAuthorizationMiddleware_NoHeaderRequest(t *testing.T) {
	router := setUpRouter()
	protectedPaths := []struct {
		name   string
		method string
		url    string
	}{
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

func TestAdminOrUserAuthorizationMiddleware_TokenValidation(t *testing.T) {
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

func TestGetUserAccounts_AdminOnlyAuthorizationMiddleware(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		role         int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "AdminUser_GetUserAccounts",
			username:     "janedoe",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_GetUserAccounts",
			username:     "someuser",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin access required"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := auth.GenerateJWT(tc.username, tc.role, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := bearer + jwtToken
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

func TestGetUserAccount_AdminOrMeAuthorizationMiddleware(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		role         int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_GetOwnUserAccount",
			username:     "janedoe",
			role:         configmodels.UserRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "AdminUser_GetOwnUserAccount",
			username:     "janedoe",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_GetOtherUserAccount",
			username:     "someuser",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin or me access required"}`,
		},
		{
			name:         "AdminUser_GetOtherUserAccount",
			username:     "someuser",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/config/v1/account/janedoe", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := auth.GenerateJWT(tc.username, tc.role, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := bearer + jwtToken
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

func TestCreateUserAccount_CreateFirstUserWithoutHeaderAuthorization(t *testing.T) {
	router := setUpMockedRouter()
	dbadapter.WebuiDBClient = &MockMongoClientEmptyDB{}
	req, err := http.NewRequest(http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	expectedCode := http.StatusOK
	expectedBody := successBody
	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if w.Body.String() != expectedBody {
		t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
	}
}

func TestCreateUserAccount_AdminAuthorizationMiddleware(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		role         int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "AdminUser_CreateUserAccount",
			username:     "janedoe",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_CreateUserAccoun",
			username:     "someuser",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin access required"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := auth.GenerateJWT(tc.username, tc.role, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := bearer + jwtToken
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

func TestDeleteUserAccount_AdminOnlyAuthorizationMiddleware(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		role         int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_DeleteOwnUserAccount",
			username:     "janedoe",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin access required"}`,
		},
		{
			name:         "AdminUser_DeleteOwnUserAccount",
			username:     "janedoe",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_DeleteOtherUserAccount",
			username:     "someuser",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin access required"}`,
		},
		{
			name:         "AdminUser_DeleteOtherUserAccount",
			username:     "someuser",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, "/config/v1/account/janedoe", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := auth.GenerateJWT(tc.username, tc.role, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := bearer + jwtToken
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

func TestChangePassword_AdminOrMeAuthorizationMiddleware(t *testing.T) {
	router := setUpMockedRouter()

	testCases := []struct {
		name         string
		username     string
		role         int
		expectedCode int
		expectedBody string
	}{
		{
			name:         "RegularUser_OwnUserAccount",
			username:     "janedoe",
			role:         configmodels.UserRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "AdminUser_OwnUserAccount",
			username:     "janedoe",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_OtherUserAccount",
			username:     "someuser",
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin or me access required"}`,
		},
		{
			name:         "AdminUser_OtherUserAccount",
			username:     "someuser",
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/config/v1/account/janedoe/change_password", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			jwtToken, err := auth.GenerateJWT(tc.username, tc.role, mockJWTSecret)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			validToken := bearer + jwtToken
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
