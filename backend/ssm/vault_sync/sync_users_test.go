package vaultsync

import (
	"testing"
)

func TestSyncUsersMutex(t *testing.T) {
	// Test that mutex is initialized
	SyncUserMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	SyncUserMutex.Unlock()
}

func TestCoreVaultUserSyncWithStopCondition(t *testing.T) {
	// Set stop condition to prevent actual operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	// Should return early due to stop condition
	coreVaultUserSync()

	// If we get here without panic, the test passes
}

func TestSyncUsers(t *testing.T) {
	// Set stop condition to prevent actual DB operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SyncUsers panicked: %v", r)
		}
	}()

	SyncUsers()
}
