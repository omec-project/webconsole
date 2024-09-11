// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/dbadapter"
)

var protectedPaths = []struct {
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

func setUp() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	dbadapter.UserAccountDBClient = &MockMongoClientSuccess{}
	mockJWTSecret := []byte("mockSecret")
	router.Use(AuthMiddleware(mockJWTSecret))
	AddService(router, mockJWTSecret)
	configapi.AddServiceSub(router)
	configapi.AddService(router)
	return router
}

func TestMiddleware_NoHeaderRequest(t *testing.T) {
	router := setUp()

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.url, nil)
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

func TestMiddleware_InvalidHeaderRequest(t *testing.T) {
	router := setUp()

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.url, nil)
			invalidHeader := "Bearer"
			req.Header.Set("Authorization", invalidHeader)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			expectedBody := `{"error":"auth failed: authorization header couldn't be processed. The expected format is 'Bearer token'"}`
			if expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != expectedBody {
				t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
			}
		})
	}
}

func TestMiddleware_InvalidTokenRequest(t *testing.T) {
	router := setUp()

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.url, nil)
			invalidHeader := "Bearer mytoken"
			req.Header.Set("Authorization", invalidHeader)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			expectedCode := http.StatusUnauthorized
			expectedBody := `{"error":"auth failed: token is not valid"}`
			if expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
			}
			if w.Body.String() != expectedBody {
				t.Errorf("Expected `%v`, got `%v`", expectedBody, w.Body.String())
			}
		})
	}
}
