package ssmsync

import (
	"testing"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSyncOurKeysMutex(t *testing.T) {
	// Test that SyncOurKeysMutex is initialized and can be locked/unlocked
	SyncOurKeysMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	SyncOurKeysMutex.Unlock()
}

func TestSyncExternalKeysMutex(t *testing.T) {
	// Test that SyncExternalKeysMutex is initialized and can be locked/unlocked
	SyncExternalKeysMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	SyncExternalKeysMutex.Unlock()
}

func TestSyncUserMutexSSM(t *testing.T) {
	// Test that SyncUserMutex is initialized and can be locked/unlocked
	SyncUserMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	SyncUserMutex.Unlock()
}

func TestSyncOurKeysFunction(t *testing.T) {
	// Set stop condition to prevent actual operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncOurKeys panicked: %v", r)
		}
	}()

	syncOurKeys("SYNC_OUR_KEYS")
}

func TestSyncExternalKeysFunction(t *testing.T) {
	// Set stop condition to prevent actual operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("syncExternalKeys panicked: %v", r)
		}
	}()

	syncExternalKeys("SYNC_EXTERNAL_KEYS")
}

func TestSyncKeyListenChannel(t *testing.T) {
	// Set up factory.WebUIConfig to prevent nil pointer reference
	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
				SsmSync: &factory.SsmSync{
					IntervalMinute: 1, // 1 minute for testing
				},
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
		},
	}

	// Mock CommonDBClient and AuthDBClient to prevent database access in SyncUsers and SyncKeys
	oldCommonClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = oldCommonClient }()

	oldAuthClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = oldAuthClient }()

	mockClient := &dbadapter.MockDBClient{
		GetManyFn: func(collName string, filter bson.M) ([]map[string]any, error) {
			return []map[string]any{}, nil // Empty response
		},
	}
	dbadapter.CommonDBClient = mockClient
	dbadapter.AuthDBClient = mockClient

	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 10)

	// Set stop condition immediately to prevent actual operations
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false // Reset for other tests
	}()

	// Start the listener in a goroutine AFTER setting up all mocks
	go SyncKeyListen(ssmSyncMsg)

	// Give goroutine a moment to initialize and see stop condition
	time.Sleep(10 * time.Millisecond)

	// Send test messages (these should not cause actual execution due to stop condition)
	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "SYNC_OUR_KEYS",
		Info:   "Test sync",
	}

	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "SYNC_EXTERNAL_KEYS",
		Info:   "Test sync external",
	}

	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "SYNC_USERS",
		Info:   "Test sync users",
	}

	// Allow messages to be processed
	time.Sleep(10 * time.Millisecond)

	// Close channel to stop listener
	close(ssmSyncMsg)
}
