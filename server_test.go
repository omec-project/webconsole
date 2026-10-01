// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/nfconfig"
	"github.com/omec-project/webconsole/backend/webui_service"
	"github.com/urfave/cli/v3"
)

const (
	cliAppName     = "webconsole"
	cfgFlag        = "-cfg"
	testConfigFile = "test.conf"
)

type mockWebUI struct {
	started     atomic.Bool
	startedCh   chan struct{}
	startedOnce sync.Once
}

func newMockWebUI() *mockWebUI {
	return &mockWebUI{startedCh: make(chan struct{})}
}

func (m *mockWebUI) Start(ctx context.Context, syncChan chan<- struct{}) {
	m.started.Store(true)
	m.startedOnce.Do(func() {
		close(m.startedCh)
	})
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

type mockNFConfig struct{}

func (m *mockNFConfig) Start(ctx context.Context, syncChan <-chan struct{}) error {
	return nil
}

func TestRunWebUIAndNFConfig_Success_ExpectNoError(t *testing.T) {
	webui := newMockWebUI()
	nf := &mockNFConfigSuccess{}

	err := runWebUIAndNFConfig(webui, nf)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	select {
	case <-webui.startedCh:
	case <-time.After(time.Second):
		t.Fatal("webui.Start was not called in time")
	}

	if !webui.started.Load() {
		t.Errorf("webui.Start was not called in time")
	}
}

func TestRunWebUIAndNFConfig_GivenFailureInNfConfigServiceExpectError(t *testing.T) {
	webui := newMockWebUI()
	nf := &mockNFConfigFail{}

	err := runWebUIAndNFConfig(webui, nf)
	if err == nil || !strings.Contains(err.Error(), "NFConfig start failed") {
		t.Errorf("expected NFConfig failure, got %v", err)
	}
}

// mockBlockingWebUI and mockBlockingNFConfig run until the context is
// cancelled, as the real servers do.
type mockBlockingWebUI struct {
	startedCh chan struct{}
	stopped   atomic.Bool
}

func (m *mockBlockingWebUI) Start(ctx context.Context, syncChan chan<- struct{}) {
	close(m.startedCh)
	<-ctx.Done()
	m.stopped.Store(true)
}

type mockBlockingNFConfig struct{}

func (m *mockBlockingNFConfig) Start(ctx context.Context, syncChan <-chan struct{}) error {
	<-ctx.Done()
	return nil
}

func TestRunWebUIAndNFConfig_GivenSIGTERM_ExpectBothStoppedAndNoError(t *testing.T) {
	webui := &mockBlockingWebUI{startedCh: make(chan struct{})}
	errCh := make(chan error, 1)
	go func() { errCh <- runWebUIAndNFConfig(webui, &mockBlockingNFConfig{}) }()

	select {
	case <-webui.startedCh:
	case <-time.After(time.Second):
		t.Fatal("webui.Start was not called in time")
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("sending SIGTERM: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected no error after SIGTERM, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runWebUIAndNFConfig did not return after SIGTERM")
	}
	if !webui.stopped.Load() {
		t.Error("expected the WebUI to have stopped before runWebUIAndNFConfig returned")
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
			args:        []string{cliAppName},
			expectError: true,
		},
		{
			name:        "valid config flag",
			args:        []string{cliAppName, cfgFlag, testConfigFile},
			expectError: false,
		},
		{
			name:        "empty config value",
			args:        []string{cliAppName, cfgFlag, ""},
			expectError: true,
		},
		{
			name:        "invalid flag",
			args:        []string{cliAppName, "-invalid", testConfigFile},
			expectError: true,
		},
		{
			name:        "multiple flags with valid config",
			args:        []string{cliAppName, cfgFlag, testConfigFile, "-verbose"},
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
	originalEnsure := ensureSubscriberIndexes
	originalNewNF := newNFConfigServer
	originalRun := runServer
	defer func() {
		initMongoDB = originalInit
		ensureSubscriberIndexes = originalEnsure
		newNFConfigServer = originalNewNF
		runServer = originalRun
	}()
	ensureSubscriberIndexes = func() error { return nil }

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

	t.Run("index ensure failure", func(t *testing.T) {
		initMongoDB = func() error { return nil }
		ensureSubscriberIndexes = func() error { return fmt.Errorf("index failed") }
		defer func() { ensureSubscriberIndexes = func() error { return nil } }()
		newNFConfigServer = func(config *factory.Config) (nfconfig.NFConfigInterface, error) {
			t.Error("NF config server created although the indexes failed")
			return nil, fmt.Errorf("unreachable")
		}
		err := startApplication(&factory.Config{Configuration: &factory.Configuration{}})
		if err == nil || !strings.Contains(err.Error(), "index failed") {
			t.Errorf("expected index ensure error, got: %v", err)
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
			return &mockNFConfig{}, nil
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
			return &mockNFConfig{}, nil
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
