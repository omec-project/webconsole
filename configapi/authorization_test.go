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
	deviceGroupSomeNamePath   = "/config/v1/device-group/some-name"
	networkSliceSomeSlicePath = "/config/v1/network-slice/some-slice"
	subscriberSomeSubsPath    = "/api/subscriber/some-subs"
	successBody               = `{"Result":"Operation Executed"}`
	forbiddenAdminBody        = `{"error":"forbidden: admin access required"}`
	bearer                    = "Bearer "
	testUserSomeuser          = "someuser"
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
			url:    deviceGroupSomeNamePath,
		},
		{
			name:   "DeviceGroupGroupNameDelete",
			method: http.MethodDelete,
			url:    deviceGroupSomeNamePath,
		},
		{
			name:   "DeviceGroupGroupNamePatch",
			method: http.MethodPatch,
			url:    deviceGroupSomeNamePath,
		},
		{
			name:   "DeviceGroupGroupNamePut",
			method: http.MethodPut,
			url:    deviceGroupSomeNamePath,
		},
		{
			name:   "DeviceGroupGroupNamePost",
			method: http.MethodPost,
			url:    deviceGroupSomeNamePath,
		},
		{
			name:   "GetNetworkSlices",
			method: http.MethodGet,
			url:    "/config/v1/network-slice",
		},
		{
			name:   "GetNetworkSliceByName",
			method: http.MethodGet,
			url:    networkSliceSomeSlicePath,
		},
		{
			name:   "NetworkSliceSliceNameDelete",
			method: http.MethodDelete,
			url:    networkSliceSomeSlicePath,
		},
		{
			name:   "NetworkSliceSliceNamePost",
			method: http.MethodPost,
			url:    networkSliceSomeSlicePath,
		},
		{
			name:   "NetworkSliceSliceNamePut",
			method: http.MethodPut,
			url:    networkSliceSomeSlicePath,
		},
		{
			name:   "GetGnbs",
			method: http.MethodGet,
			url:    gnbInventoryPath,
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
			url:    upfInventoryPath,
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
			url:    subscriberPath,
		},
		{
			name:   "GetSubscriberByID",
			method: http.MethodGet,
			url:    subscriberSomeSubsPath,
		},
		{
			name:   "PostSubscriberByID",
			method: http.MethodPost,
			url:    subscriberSomeSubsPath,
		},
		{
			name:   "PutSubscriberByID",
			method: http.MethodPut,
			url:    "/api/subscriber/some-subs/plmnid",
		},
		{
			name:   "DeleteSubscriberByID",
			method: http.MethodDelete,
			url:    subscriberSomeSubsPath,
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
			req, err := http.NewRequestWithContext(t.Context(), tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			expectedBody := `{"error":"auth failed: authorization header not found"}`
			if expectedCode != w.Code {
				t.Errorf("expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != expectedBody {
				t.Errorf("expected `%v`, got `%v`", expectedBody, w.Body.String())
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
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/config/v1/", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			if expectedCode != w.Code {
				t.Errorf("expected status code `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected body `%v`, got `%v`", tc.expectedBody, w.Body.String())
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
			username:     testUserJanedoe,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_GetUserAccounts",
			username:     testUserSomeuser,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: forbiddenAdminBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/config/v1/account", nil)
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
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
			username:     testUserJanedoe,
			role:         configmodels.UserRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "AdminUser_GetOwnUserAccount",
			username:     testUserJanedoe,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_GetOtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin or me access required"}`,
		},
		{
			name:         "AdminUser_GetOtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/config/v1/account/janedoe", nil)
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestCreateUserAccount_CreateFirstUserWithoutHeaderAuthorization(t *testing.T) {
	router := setUpMockedRouter()
	dbadapter.WebuiDBClient = &MockMongoClientEmptyDB{}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	expectedCode := http.StatusOK
	expectedBody := successBody
	if expectedCode != w.Code {
		t.Errorf("expected `%v`, got `%v`", expectedCode, w.Code)
	}
	if w.Body.String() != expectedBody {
		t.Errorf("expected `%v`, got `%v`", expectedBody, w.Body.String())
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
			username:     testUserJanedoe,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_CreateUserAccoun",
			username:     testUserSomeuser,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: forbiddenAdminBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/account", strings.NewReader(`{"username": "adminadmin", "password":"ValidPass123!"}`))
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
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
			username:     testUserJanedoe,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: forbiddenAdminBody,
		},
		{
			name:         "AdminUser_DeleteOwnUserAccount",
			username:     testUserJanedoe,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_DeleteOtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: forbiddenAdminBody,
		},
		{
			name:         "AdminUser_DeleteOtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/config/v1/account/janedoe", nil)
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
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
			username:     testUserJanedoe,
			role:         configmodels.UserRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "AdminUser_OwnUserAccount",
			username:     testUserJanedoe,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
		{
			name:         "RegularUser_OtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.UserRole,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":"forbidden: admin or me access required"}`,
		},
		{
			name:         "AdminUser_OtherUserAccount",
			username:     testUserSomeuser,
			role:         configmodels.AdminRole,
			expectedCode: http.StatusOK,
			expectedBody: successBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/config/v1/account/janedoe/change_password", nil)
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
				t.Errorf("expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if w.Body.String() != tc.expectedBody {
				t.Errorf("expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}
