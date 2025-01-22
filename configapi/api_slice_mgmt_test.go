// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeviceGroupPostHandler_InvalidDeviceGroupName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	route := "/config/v1/device-group/invalid&name"
	header := "application/json"

	req, err := http.NewRequest(http.MethodPost, route, strings.NewReader(`"whatever": "json"`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", header)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	expectedCode := http.StatusBadRequest
	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
}

func TestNetworkSlicePostHandler_InvalidNetworkSliceName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	route := "/config/v1/network-slice/invalid&name"
	header := "application/json"

	req, err := http.NewRequest(http.MethodPost, route, strings.NewReader(`"whatever": "json"`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", header)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	expectedCode := http.StatusBadRequest
	if expectedCode != w.Code {
		t.Errorf("Expected `%v`, got `%v`", expectedCode, w.Code)
	}
}
