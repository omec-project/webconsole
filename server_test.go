// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/urfave/cli"
)

type mockWebUI struct {
	started bool
}

func (m *mockWebUI) Start(ctx context.Context) {
	m.started = true
}

type mockNFConfigSuccess struct{}

func (m *mockNFConfigSuccess) Start(ctx context.Context) error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

type mockNFConfigFail struct{}

func (m *mockNFConfigFail) Start(ctx context.Context) error {
	return errors.New("NFConfig start failed")
}

func TestRunWebUIAndNFConfig_Success(t *testing.T) {
	webui := &mockWebUI{}
	nf := &mockNFConfigSuccess{}

	err := runWebUIAndNFConfig(webui, nf)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !webui.started {
		t.Errorf("expected webui to be started")
	}
}

func TestRunWebUIAndNFConfig_Failure(t *testing.T) {
	webui := &mockWebUI{}
	nf := &mockNFConfigFail{}

	err := runWebUIAndNFConfig(webui, nf)
	if err == nil || err.Error() != "NFConfig failed: NFConfig start failed" {
		t.Errorf("expected NFConfig failure, got %v", err)
	}
	if !webui.started {
		t.Errorf("expected webui should not start when nf fails")
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
			app := cli.NewApp()
			app.Name = "webui"
			app.Usage = "Web UI"
			app.UsageText = "webconsole -cfg <webui_config_file.yaml>"
			app.Flags = factory.GetCliFlags()
			app.Action = func(c *cli.Context) error {
				cfg := c.String("cfg")
				if cfg == "" {
					return fmt.Errorf("required flag cfg not set")
				}
				return nil
			}
			err := app.Run(tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
