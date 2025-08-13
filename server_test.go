// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli/v3"
)

type mockWebUI struct {
	started     bool
	startedChan chan struct{}
}

func (m *mockWebUI) Start(ctx context.Context, syncChan chan<- struct{}) {
	select {
	case <-ctx.Done():
		return
	default:
		m.started = true
		if m.startedChan != nil {
			close(m.startedChan)
		}
	}
}

type mockNFConfigSuccess struct{}

func (m *mockNFConfigSuccess) Start(ctx context.Context, syncChan <-chan struct{}) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

type mockNFConfigFail struct{}

func (m *mockNFConfigFail) Start(ctx context.Context, syncChan <-chan struct{}) error {
	return errors.New("NFConfig start failed")
}

type MockNFConfig struct{}

func (m *MockNFConfig) Start(ctx context.Context, syncChan <-chan struct{}) error {
	return nil
}

func TestRunWebUIAndNFConfig_Success(t *testing.T) {
	started := make(chan struct{})
	webui := &mockWebUI{startedChan: started}
	nf := &mockNFConfigSuccess{}

	err := runWebUIAndNFConfig(webui, nf)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Errorf("webui.Start was not called in time")
	}
}

func TestRunWebUIAndNFConfig_Failure(t *testing.T) {
	started := make(chan struct{})
	webui := &mockWebUI{startedChan: started}
	nf := &mockNFConfigFail{}

	err := runWebUIAndNFConfig(webui, nf)
	if err == nil || !strings.Contains(err.Error(), "NFConfig start failed") {
		t.Errorf("expected NFConfig failure, got %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	if webui.started {
		t.Errorf("webui.Start() should respect context cancellation and not proceed")
	}
}

func TestMainValidateCLIFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "missing required flag",
			args:        []string{"webconsole"},
			expectError: true,
		},
		{
			name:        "valid config flag",
			args:        []string{"webconsole", "-cfg", "test.conf"},
			expectError: false,
		},
		{
			name:        "empty config value",
			args:        []string{"webconsole", "-cfg", ""},
			expectError: true,
		},
		{
			name:        "invalid flag",
			args:        []string{"webconsole", "-invalid", "test.conf"},
			expectError: true,
		},
		{
			name:        "multiple flags with valid config",
			args:        []string{"webconsole", "-cfg", "test.conf", "-verbose"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.Command{}
			app.Name = "webui"
			app.Usage = "Web UI"
			app.UsageText = "webconsole -cfg <webui_config_file.yaml>"
			app.Flags = factory.GetCliFlags()
			app.Action = func(ctx context.Context, c *cli.Command) error {
				cfg := c.String("cfg")
				if cfg == "" {
					return fmt.Errorf("required flag cfg not set")
				}
				return nil
			}
			err := app.Run(context.Background(), tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStartApplication(t *testing.T) {
	originalInit := initMongoDB
	originalNewNF := newNFConfigServer
	originalRun := runServer
	defer func() {
		initMongoDB = originalInit
		newNFConfigServer = originalNewNF
		runServer = originalRun
	}()

	t.Run("nil config", func(t *testing.T) {
		err := startApplication(nil)
		if err == nil || !strings.Contains(err.Error(), "nil") {
			t.Errorf("expected error for nil config, got: %v", err)
		}
	})

	t.Run("mongo init failure", func(t *testing.T) {
		initMongoDB = func() error {
			return fmt.Errorf("mongo failed")
		}
		err := startApplication(&factory.Config{Configuration: &factory.Configuration{}})
		if err == nil || !strings.Contains(err.Error(), "mongo failed") {
			t.Errorf("expected mongo init error, got: %v", err)
		}
	})

	t.Run("nfconfig init failure", func(t *testing.T) {
		initMongoDB = func() error { return nil }
		newNFConfigServer = func(config *factory.Config) (nfconfig.NFConfigInterface, error) {
			return nil, fmt.Errorf("nfconfig init fail")
		}
		err := startApplication(&factory.Config{Configuration: &factory.Configuration{}})
		if err == nil || !strings.Contains(err.Error(), "nfconfig init fail") {
			t.Errorf("expected NF config init failure, got: %v", err)
		}
	})

	t.Run("run failure", func(t *testing.T) {
		initMongoDB = func() error { return nil }
		newNFConfigServer = func(config *factory.Config) (nfconfig.NFConfigInterface, error) {
			return &MockNFConfig{}, nil
		}
		runServer = func(webui webui_service.WebUIInterface, nf nfconfig.NFConfigInterface) error {
			return fmt.Errorf("run fail")
		}
		err := startApplication(&factory.Config{Configuration: &factory.Configuration{}})
		if err == nil || !strings.Contains(err.Error(), "run fail") {
			t.Errorf("expected run error, got: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		initMongoDB = func() error { return nil }
		newNFConfigServer = func(config *factory.Config) (nfconfig.NFConfigInterface, error) {
			return &MockNFConfig{}, nil
		}
		runServer = func(webui webui_service.WebUIInterface, nf nfconfig.NFConfigInterface) error {
			return nil
		}
		err := startApplication(&factory.Config{Configuration: &factory.Configuration{}})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}
