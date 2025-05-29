// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/factory"
)

func TestNewNFConfig_nil_config(t *testing.T) {
	_, err := NewNFConfig(nil)
	if err == nil {
		t.Errorf("expected error for nil config, got nil.")
	}
}

func TestNewNFConfig_various_configs(t *testing.T) {
	testCases := []struct {
		name        string
		config      *factory.Config
		expectError bool
	}{
		{
			name: "correct TLS configuration and warning log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "warning",
					},
				},
				Configuration: &factory.Configuration{
					ConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
			expectError: false,
		},
		{
			name: "correct TLS configuration and info log level",
			config: &factory.Config{
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
				},
			},
			expectError: false,
		},
		{
			name: "missing key and error log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "info",
					},
				},
				Configuration: &factory.Configuration{
					ConfigTLS: &factory.TLS{
						PEM: "test.pem",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing pem and debug log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "info",
					},
				},
				Configuration: &factory.Configuration{
					ConfigTLS: &factory.TLS{
						Key: "test.key",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid debug level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "invalid_level",
					},
				},
				Configuration: &factory.Configuration{
					ConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nf, err := NewNFConfig(tc.config)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error for invalid config, got nil")
				}
				if nf != nil {
					t.Errorf("expected nil NFConfigInterface for invalid config, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if nf == nil {
					t.Errorf("expected non-nil NFConfigInterface, got nil")
				}
			}
		})
	}
}

func TestNFConfigRoutes_success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockValidConfig := &factory.Config{
		Logger: &logger.Logger{
			WEBUI: &logger.LogSetting{
				DebugLevel: "debug",
			},
		},
		Configuration: &factory.Configuration{
			ConfigTLS: &factory.TLS{
				PEM: "test.pem",
				Key: "test.key",
			},
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

	testCases := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "access mobility endpoint",
			path:       "/nfconfig/access-mobility",
			wantStatus: http.StatusOK,
		},
		{
			name:       "plmn endpoint",
			path:       "/nfconfig/plmn",
			wantStatus: http.StatusOK,
		},
		{
			name:       "plmn-snssai endpoint",
			path:       "/nfconfig/plmn-snssai",
			wantStatus: http.StatusOK,
		},
		{
			name:       "policy control endpoint",
			path:       "/nfconfig/policy-control",
			wantStatus: http.StatusOK,
		},
		{
			name:       "session management endpoint",
			path:       "/nfconfig/session-management",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			nf.Router().ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}

}
