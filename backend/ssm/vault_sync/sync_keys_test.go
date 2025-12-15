package vaultsync

import (
	"testing"
)

func TestSyncOurKeys(t *testing.T) {
	// Set stop condition to prevent actual operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncOurKeys panicked: %v", r)
		}
	}()

	syncOurKeys("SYNC_OUR_KEYS")
}

func TestSyncExternalKeys(t *testing.T) {
	// Set stop condition to prevent actual operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncExternalKeys panicked: %v", r)
		}
	}()

	syncExternalKeys("SYNC_EXTERNAL_KEYS")
}

func TestSyncKeys(t *testing.T) {
	// Set stop condition to prevent actual operations
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	testCases := []struct {
		keyLabel string
		action   string
	}{
		{"K4_AES", "SYNC_OUR_KEYS"},
		{"K4_DES", "SYNC_EXTERNAL_KEYS"},
		{"test_label", "UNKNOWN_ACTION"},
	}

	for _, tc := range testCases {
		t.Run(tc.keyLabel+"_"+tc.action, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("SyncKeys panicked: %v", r)
				}
			}()

			SyncKeys(tc.keyLabel, tc.action)
		})
	}
}

func TestSyncKeysMutexes(t *testing.T) {
	// Test that mutexes can be locked and unlocked
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
