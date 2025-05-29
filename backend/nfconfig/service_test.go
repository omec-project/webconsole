// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/factory"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewNFConfig_nil_config(t *testing.T) {
	_, err := NewNFConfig(nil)
	if err == nil {
		t.Errorf("expected error for nil config, got nil.")
	}
}

func TestNewNFConfig_valid_config(t *testing.T) {
	mockValidConfig := &factory.Config{
		Logger: &logger.Logger{
			WEBUI: &logger.LogSetting{
				DebugLevel: "info",
			},
		},
		Configuration: &factory.Configuration{
			ConfigTLS: &factory.TLS{
				PEM: "test.pem",
				Key: "test.key",
			},
			CfgPort: 9090,
		},
	}
	nf, err := NewNFConfig(mockValidConfig)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if nf == nil {
		t.Errorf("expected NFConfigInterface, got nil")
	}
}

func TestNFConfigRoutes_success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockValidConfig := &factory.Config{
		Logger: &logger.Logger{
			WEBUI: &logger.LogSetting{
				DebugLevel: "info",
			},
		},
		Configuration: &factory.Configuration{
			ConfigTLS: &factory.TLS{
				PEM: "test.pem",
				Key: "test.key",
			},
			CfgPort: 9090,
		},
	}

	nfInterface, err := NewNFConfig(mockValidConfig)
	if err != nil {
		t.Fatalf("failed to initialize NFConfig: %v", err)
	}

	nf, ok := nfInterface.(*NFConfig)
	if !ok {
		t.Fatalf("expected *NFConfig type")
	}

	req1, _ := http.NewRequest("GET", "/nfconfig/access-mobility", nil)
	w1 := httptest.NewRecorder()
	nf.Router().ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("GET", "/nfconfig/plmn", nil)
	w2 := httptest.NewRecorder()
	nf.Router().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w2.Code)
	}

	req3, _ := http.NewRequest("GET", "/nfconfig/plmn-snssai", nil)
	w3 := httptest.NewRecorder()
	nf.Router().ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w3.Code)
	}

	req4, _ := http.NewRequest("GET", "/nfconfig/policy-control", nil)
	w4 := httptest.NewRecorder()
	nf.Router().ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w4.Code)
	}

	req5, _ := http.NewRequest("GET", "/nfconfig/session-management", nil)
	w5 := httptest.NewRecorder()
	nf.Router().ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w5.Code)
	}
}
