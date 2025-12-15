package vaultsync

import (
	"testing"
)

func TestReadStopCondition(t *testing.T) {
	// Set initial condition
	setStopCondition(false)

	result := readStopCondition()
	if result != false {
		t.Errorf("Expected readStopCondition() to return false, got %v", result)
	}

	// Change condition
	setStopCondition(true)
	result = readStopCondition()
	if result != true {
		t.Errorf("Expected readStopCondition() to return true, got %v", result)
	}

	// Reset for other tests
	setStopCondition(false)
}

func TestSetStopCondition(t *testing.T) {
	setStopCondition(true)
	if !readStopCondition() {
		t.Error("setStopCondition(true) should set the flag to true")
	}

	setStopCondition(false)
	if readStopCondition() {
		t.Error("setStopCondition(false) should set the flag to false")
	}
}

// func TestErrorSyncChanInitialized(t *testing.T) {
// 	if ErrorSyncChan == nil {
// 		t.Error("ErrorSyncChan should be initialized")
// 	}

// 	// Test that we can send to the channel without blocking
// 	select {
// 	case ErrorSyncChan <- nil:
// 		// Successfully sent
// 	default:
// 		t.Error("ErrorSyncChan should accept messages")
// 	}

// 	// Drain the channel
// 	select {
// 	case <-ErrorSyncChan:
// 		// Successfully received
// 	default:
// 		t.Error("Should have been able to receive from ErrorSyncChan")
// 	}
// }

// func TestErrorSyncChanCapacity(t *testing.T) {
// 	if cap(ErrorSyncChan) != 10 {
// 		t.Errorf("Expected ErrorSyncChan capacity of 10, got %d", cap(ErrorSyncChan))
// 	}
// }

func TestStopVaultSyncFunctionInitialValue(t *testing.T) {
	// Reset to known state
	setStopCondition(false)

	if readStopCondition() != false {
		t.Error("StopVaultSyncFunction should be initialized to false")
	}
}

func TestConstants(t *testing.T) {
	if internalKeyLabel != "aes256-gcm" {
		t.Errorf("Expected internalKeyLabel to be 'aes256-gcm', got '%s'", internalKeyLabel)
	}

	if getTransitKeysListPath() != "transit/keys" {
		t.Errorf("Expected getTransitKeysListPath() to return 'transit/keys', got '%s'", getTransitKeysListPath())
	}

	if getTransitKeyCreateFormat() != "transit/keys/%s" {
		t.Errorf("Expected getTransitKeyCreateFormat() to return 'transit/keys/%%s', got '%s'", getTransitKeyCreateFormat())
	}

	if getExternalKeysListPath() != "secret/metadata/k4keys" {
		t.Errorf("Expected getExternalKeysListPath() to return 'secret/metadata/k4keys', got '%s'", getExternalKeysListPath())
	}
}

func TestConcurrentStopConditionAccess(t *testing.T) {
	// Test concurrent access to stop condition
	done := make(chan bool)

	// Start multiple goroutines reading and writing
	for i := 0; i < 10; i++ {
		go func(val bool) {
			setStopCondition(val)
			_ = readStopCondition()
			done <- true
		}(i%2 == 0)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Reset to known state
	setStopCondition(false)
}
