// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
)

func TestNewNFConfig_nil_config(t *testing.T) {
	_, err := NewNFConfigServer(nil)
	if err == nil {
		t.Errorf("expected error for nil config, got nil.")
	}
}

func TestNewNFConfig_various_configs(t *testing.T) {
	testCases := []struct {
		name   string
		config *factory.Config
	}{
		{
			name: "correct TLS configuration and warn log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "warn",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
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
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "missing key and error log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "error",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
					},
				},
			},
		},
		{
			name: "missing pem and debug log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "debug",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						Key: "test.key",
					},
				},
			},
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
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "correct TLS configuration and wrong log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "invalid",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{},
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			nf, err := NewNFConfigServer(tc.config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if nf == nil {
				t.Errorf("expected non-nil NFConfigInterface, got nil")
			}
		})
	}
}

func TestNFConfigRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockValidConfig := &factory.Config{
		Logger: &logger.Logger{
			WEBUI: &logger.LogSetting{
				DebugLevel: "debug",
			},
		},
		Configuration: &factory.Configuration{
			NfConfigTLS: &factory.TLS{
				PEM: "test.pem",
				Key: "test.key",
			},
		},
	}

	mockDB := &MockDBClient{
		Slices: []configmodels.Slice{},
	}
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	dbadapter.CommonDBClient = mockDB

	nfInterface, err := NewNFConfigServer(mockValidConfig)
	if err != nil {
		t.Fatalf("failed to initialize NFConfig: %v", err)
	}

	nf, ok := nfInterface.(*NFConfigServer)
	if !ok {
		t.Fatalf("expected *NFConfig type")
	}

	testCases := []struct {
		name         string
		path         string
		acceptHeader string
		wantStatus   int
	}{
		{
			name:         "access mobility endpoint status OK",
			path:         "/nfconfig/access-mobility",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "plmn endpoint status OK",
			path:         "/nfconfig/plmn",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "plmn-snssai endpoint status OK",
			path:         "/nfconfig/plmn-snssai",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "policy control endpoint status OK",
			path:         "/nfconfig/policy-control",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "session management endpoint status OK",
			path:         "/nfconfig/session-management",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "access mobility endpoint invalid accept header",
			path:         "/nfconfig/access-mobility",
			acceptHeader: "",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "plmn endpoint invalid accept header",
			path:         "/nfconfig/plmn",
			acceptHeader: "json",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "plmn-snssai endpoint invalid accept header",
			path:         "/nfconfig/plmn-snssai",
			acceptHeader: "text/html",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "policy control endpoint invalid accept header",
			path:         "/nfconfig/policy-control",
			acceptHeader: "text/html",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "session management endpoint invalid accept header",
			path:         "/nfconfig/session-management",
			acceptHeader: "application/jsons",
			wantStatus:   http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			req.Header.Set("Accept", tc.acceptHeader)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			nf.router().ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestNFConfigStart(t *testing.T) {
	tests := []struct {
		name    string
		config  *factory.Configuration
		wantErr bool
	}{
		{
			name: "HTTP server start and graceful shutdown",
			config: &factory.Configuration{
				NfConfigTLS: nil,
			},
			wantErr: false,
		},
		{
			name: "HTTPS server start and graceful shutdown",
			config: &factory.Configuration{
				NfConfigTLS: &factory.TLS{
					PEM: "testdata/test.pem",
					Key: "testdata/test.key",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Starting test: %s", tt.name)
			gin.SetMode(gin.TestMode)
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{},
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			nfconf := &NFConfigServer{
				config: tt.config,
				Router: gin.New(),
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			errChan := make(chan error, 1)

			go func() {
				t.Logf("starting server")
				err := nfconf.Start(ctx)
				t.Logf("server stopped with error: %v", err)
				errChan <- err
			}()

			time.Sleep(500 * time.Millisecond)
			t.Logf("triggering shutdown")
			cancel()

			select {
			case err := <-errChan:
				if tt.wantErr && err == nil {
					t.Errorf("got error = nil, wantErr %v", tt.wantErr)
				}
				if !tt.wantErr && err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("test timed out waiting for server to stop")
			}
		})
	}
}

func TestNFConfig_Start_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc1 := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	nfc2 := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	errChan := make(chan error, 1)
	go func() {
		errChan <- nfc1.Start(ctx1)
	}()
	time.Sleep(10 * time.Millisecond)

	ctx2 := context.Background()
	err := nfc2.Start(ctx2)
	if err == nil {
		t.Error("expected error when starting server on same port, got nil")
	}
	cancel1()
	<-errChan
}

func TestNFConfig_Start_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	go func() {
		errChan <- nfc.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	t.Logf("triggering context cancellation")
	cancel()
	err := <-errChan
	if err != nil {
		t.Errorf("got error = %v, want nil after context cancellation", err)
	}
}
