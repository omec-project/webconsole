package ssm

import (
	"testing"
)

// Mock implementation of SSM interface for testing
type MockSSM struct {
	LoginCalled             bool
	HealthCheckCalled       bool
	SyncKeyListenCalled     bool
	KeyRotationListenCalled bool
	InitDefaultCalled       bool
	LoginError              error
	LoginToken              string
	InitDefaultError        error
}

func (m *MockSSM) SyncKeyListen(chan *SsmSyncMessage) {
	m.SyncKeyListenCalled = true
}

func (m *MockSSM) KeyRotationListen(chan *SsmSyncMessage) {
	m.KeyRotationListenCalled = true
}

func (m *MockSSM) Login() (string, error) {
	m.LoginCalled = true
	return m.LoginToken, m.LoginError
}

func (m *MockSSM) HealthCheck() {
	m.HealthCheckCalled = true
}

func (m *MockSSM) InitDefault(ssmSyncMsg chan *SsmSyncMessage) error {
	m.InitDefaultCalled = true
	return m.InitDefaultError
}

func TestSsmSyncMessage(t *testing.T) {
	msg := SsmSyncMessage{
		Action: "TEST_ACTION",
		Info:   "Test information",
	}

	if msg.Action != "TEST_ACTION" {
		t.Errorf("Expected Action to be 'TEST_ACTION', got '%s'", msg.Action)
	}

	if msg.Info != "Test information" {
		t.Errorf("Expected Info to be 'Test information', got '%s'", msg.Info)
	}
}

func TestMockSSMLogin(t *testing.T) {
	mock := &MockSSM{
		LoginToken: "test-token-123",
	}

	token, err := mock.Login()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", token)
	}

	if !mock.LoginCalled {
		t.Error("Login should have been called")
	}
}

func TestMockSSMHealthCheck(t *testing.T) {
	mock := &MockSSM{}

	mock.HealthCheck()

	if !mock.HealthCheckCalled {
		t.Error("HealthCheck should have been called")
	}
}

func TestMockSSMSyncKeyListen(t *testing.T) {
	mock := &MockSSM{}
	ch := make(chan *SsmSyncMessage, 1)

	mock.SyncKeyListen(ch)

	if !mock.SyncKeyListenCalled {
		t.Error("SyncKeyListen should have been called")
	}
}

func TestMockSSMKeyRotationListen(t *testing.T) {
	mock := &MockSSM{}
	ch := make(chan *SsmSyncMessage, 1)

	mock.KeyRotationListen(ch)

	if !mock.KeyRotationListenCalled {
		t.Error("KeyRotationListen should have been called")
	}
}

func TestMockSSMInitDefault(t *testing.T) {
	mock := &MockSSM{}
	ch := make(chan *SsmSyncMessage, 1)

	err := mock.InitDefault(ch)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mock.InitDefaultCalled {
		t.Error("InitDefault should have been called")
	}
}
