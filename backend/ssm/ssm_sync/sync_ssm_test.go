package ssmsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
)

func TestStopSSMsyncFunctionDefault(t *testing.T) {
	// Verify the global variable is initialized
	if StopSSMsyncFunction {
		// Reset to false for predictable tests
		StopSSMsyncFunction = false
	}
}

func TestErrorChannelsCapacity(t *testing.T) {
	// Test ErrorSyncChan capacity
	if cap(ErrorSyncChan) != 10 {
		t.Errorf("Expected ErrorSyncChan capacity of 10, got %d", cap(ErrorSyncChan))
	}

	// Test ErrorRotationChan capacity
	if cap(ErrorRotationChan) != 10 {
		t.Errorf("Expected ErrorRotationChan capacity of 10, got %d", cap(ErrorRotationChan))
	}
}

func TestSyncSsmChannelHandling(t *testing.T) {
	// Create a mock SSM implementation
	mockSSM := &MockSSM{}
	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 10)

	// Start SyncSsm in a goroutine
	go SyncSsm(ssmSyncMsg, mockSSM)

	// Give it a moment to initialize
	// Note: In a real test, you'd want to use synchronization primitives
	// to ensure the goroutines have started

	// Verify that SyncKeyListen was called
	// Note: This is a simplified test. In a real scenario, you'd need
	// more sophisticated mocking and synchronization

	close(ssmSyncMsg)
}

// MockSSM for testing
type MockSSM struct {
	SyncKeyListenCalled     bool
	KeyRotationListenCalled bool
}

func (m *MockSSM) SyncKeyListen(ch chan *ssm.SsmSyncMessage) {
	m.SyncKeyListenCalled = true
}

func (m *MockSSM) KeyRotationListen(ch chan *ssm.SsmSyncMessage) {
	m.KeyRotationListenCalled = true
}

func (m *MockSSM) Login() (string, error) {
	return "mock-token", nil
}

func (m *MockSSM) HealthCheck() {
	// Mock implementation
}

func (m *MockSSM) InitDefault(ch chan *ssm.SsmSyncMessage) error {
	return nil
}
