// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/dbadapter"
)

func TestStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	mockJWTSecret := []byte("mockSecret")
	AddAuthenticationService(router, mockJWTSecret)

	testCases := []struct {
		name         string
		dbAdapter    dbadapter.DBInterface
		expectedCode int
		expectedBody string
	}{
		{
			name:         "StatusIsInitialized",
			dbAdapter:    &MockMongoClientSuccess{},
			expectedCode: http.StatusOK,
			expectedBody: `{"initialized":true}`,
		},
		{
			name:         "StatusIsNotInitialized",
			dbAdapter:    &MockMongoClientEmptyDB{},
			expectedCode: http.StatusOK,
			expectedBody: `{"initialized":false}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbadapter.WebuiDBClient = tc.dbAdapter
			req, err := http.NewRequest(http.MethodGet, "/status", nil)
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
