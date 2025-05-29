package main

import (
	"errors"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/urfave/cli"
	"testing"
	"time"
)

type mockWebUI struct {
	started bool
}

func (m *mockWebUI) Initialize(c *cli.Context) (*factory.Config, error) {
	return &factory.Config{}, nil
}

func (m *mockWebUI) GetCliCmd() []cli.Flag { return nil }
func (m *mockWebUI) Start() {
	m.started = true
}

type mockNFConfigSuccess struct{}

func (m *mockNFConfigSuccess) Start() error {
	time.Sleep(50 * time.Millisecond)
	return nil
}

type mockNFConfigFail struct{}

func (m *mockNFConfigFail) Start() error {
	return errors.New("NFConfig start failed")
}

func TestRunWebUIAndNFConfig_Success(t *testing.T) {
	webui := &mockWebUI{}
	nf := &mockNFConfigSuccess{}

	err := runWebUIAndNFConfig(webui, nf)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if !webui.started {
		t.Errorf("expected WebUI.Start() to be called")
	}
}

func TestRunWebUIAndNFConfig_Failure(t *testing.T) {
	webui := &mockWebUI{}
	nf := &mockNFConfigFail{}

	err := runWebUIAndNFConfig(webui, nf)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
