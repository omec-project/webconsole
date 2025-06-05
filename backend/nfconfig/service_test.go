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

func TestNFConfigRoutes_success(t *testing.T) {
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

	nfInterface, err := NewNFConfigServer(mockValidConfig)
	if err != nil {
		t.Fatalf("failed to initialize NFConfig: %v", err)
	}

	nf, ok := nfInterface.(*NFConfigServer)
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
			name: "HTTP Server Start and Graceful Shutdown",
			config: &factory.Configuration{
				NfConfigTLS: nil,
			},
			wantErr: false,
		},
		{
			name: "HTTPS Server Start and Graceful Shutdown",
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
			nfconf := &NFConfigServer{
				Config: tt.config,
				Router: gin.New(),
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			errChan := make(chan error, 1)

			go func() {
				t.Logf("Starting server")
				err := nfconf.Start(ctx)
				t.Logf("Server stopped with error: %v", err)
				errChan <- err
			}()

			time.Sleep(500 * time.Millisecond)
			t.Logf("Triggering shutdown")
			cancel()

			select {
			case err := <-errChan:
				if tt.wantErr && err == nil {
					t.Errorf("Got error = nil, wantErr %v", tt.wantErr)
				}
				if !tt.wantErr && err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("Got error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("Test timed out waiting for server to stop")
			}
		})
	}
}

func TestNFConfig_Start_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc1 := &NFConfigServer{
		Config: &factory.Configuration{},
		Router: gin.New(),
	}
	nfc2 := &NFConfigServer{
		Config: &factory.Configuration{},
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
		t.Error("Expected error when starting server on same port, got nil")
	}
	cancel1()
	<-errChan
}

func TestNFConfig_Start_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc := &NFConfigServer{
		Config: &factory.Configuration{},
		Router: gin.New(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	go func() {
		errChan <- nfc.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	t.Logf("Triggering context cancellation")
	cancel()
	err := <-errChan
	if err != nil {
		t.Errorf("Got error = %v, want nil after context cancellation", err)
	}
}
