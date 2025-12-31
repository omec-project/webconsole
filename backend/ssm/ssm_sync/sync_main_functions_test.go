package ssmsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
)

func TestSsmSyncInitDefault(t *testing.T) {
	// Create a buffered channel to prevent blocking
	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 10)

	// Set stop condition to prevent actual sync operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should return early due to stop condition
	SsmSyncInitDefault(ssmSyncMsg)

	// No messages should be sent due to stop condition
	select {
	case <-ssmSyncMsg:
		t.Error("Expected no messages when StopSSMsyncFunction is true")
	default:
		// Expected behavior
	}

	close(ssmSyncMsg)
}

func TestSyncKeysWithStopCondition(t *testing.T) {
	// Set stop condition
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should return early due to stop condition
	SyncKeys("test_label", "SYNC_OUR_KEYS")

	// If we get here without panic, the test passes
}

func TestSyncUsers(t *testing.T) {
	// Set stop condition to prevent actual DB operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SyncUsers panicked: %v", r)
		}
	}()

	SyncUsers()
}

func TestCoreUserSync(t *testing.T) {
	// Set stop condition to prevent actual DB operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should return early due to stop condition
	coreUserSync()

	// If we get here without panic, the test passes
}

func TestSyncKeysActionTypes(t *testing.T) {
	// Set stop condition to prevent actual operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	testCases := []struct {
		action   string
		keyLabel string
	}{
		{"SYNC_OUR_KEYS", "K4_AES"},
		{"SYNC_EXTERNAL_KEYS", "K4_DES"},
		{"UNKNOWN_ACTION", "K4_TEST"},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			// Should not panic
			SyncKeys(tc.keyLabel, tc.action)
		})
	}
}
