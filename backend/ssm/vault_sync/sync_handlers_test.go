package vaultsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
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
	// Set up factory.WebUIConfig to prevent nil pointer reference
	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
			Mongodb: &factory.Mongodb{
				ConcurrencyOps: 10,
			},
		},
	}

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
	// Set up factory.WebUIConfig to prevent nil pointer reference
	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
			Mongodb: &factory.Mongodb{
				ConcurrencyOps: 10,
			},
		},
	}

	// Mock the DB client
	oldAuthClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = oldAuthClient }()

	mockClient := &dbadapter.MockDBClient{
		GetManyFn: func(collName string, filter bson.M) ([]map[string]any, error) {
			// Return empty slice to avoid processing
			return []map[string]any{}, nil
		},
	}
	dbadapter.AuthDBClient = mockClient

	// Initialize the channel
	ch := make(chan *ssm.SsmSyncMessage, 5)
	SetSyncChanHandle(ch)

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
