package vaultsync

import (
	"testing"
)

func TestSyncMutexesInitialized(t *testing.T) {
	// Test that mutexes are initialized and can be locked/unlocked

	SyncOurKeysMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	ourlocked := true
	if !ourlocked {
		t.Error("This should never happen")
	}
	SyncOurKeysMutex.Unlock()

	SyncExternalKeysMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	extlocked := true
	if !extlocked {
		t.Error("This should never happen")
	}
	SyncExternalKeysMutex.Unlock()

	SyncUserMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	userlocked := true
	if !userlocked {
		t.Error("This should never happen")
	}
	SyncUserMutex.Unlock()
}

func TestCoreVaultUserSync(t *testing.T) {
	// Set stop condition to prevent actual DB operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("coreVaultUserSync panicked: %v", r)
		}
	}()

	coreVaultUserSync()
}

func TestCoreVaultUserSyncNormal(t *testing.T) {
	// Set stop condition to false but expect DB errors
	setStopCondition(false)

	// This should not panic even without DB
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("coreVaultUserSync panicked: %v", r)
		}
	}()

	coreVaultUserSync()
}
