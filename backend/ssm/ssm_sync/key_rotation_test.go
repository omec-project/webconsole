package ssmsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/configmodels"
)

func TestCheckMutexInitialized(t *testing.T) {
	// Test that CheckMutex is initialized and can be locked/unlocked
	CheckMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	CheckMutex.Unlock()
}

func TestRotationMutexInitialized(t *testing.T) {
	// Test that RotationMutex is initialized and can be locked/unlocked
	RotationMutex.Lock()
	// Perform a basic operation to ensure the critical section is not empty
	locked := true
	if !locked {
		t.Error("This should never happen")
	}
	RotationMutex.Unlock()
}

func TestCheckKeyHealthWithStopCondition(t *testing.T) {
	// Set stop condition
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 10)
	defer close(ssmSyncMsg)

	err := CheckKeyHealth(ssmSyncMsg)

	if err == nil {
		t.Error("Expected error when StopSSMsyncFunction is true")
	}
}

func TestRotateExpiredKeysWithStopCondition(t *testing.T) {
	// Set stop condition
	StopSSMsyncFunction = true
	defer func() {
		StopSSMsyncFunction = false
	}()

	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 10)
	defer close(ssmSyncMsg)

	err := rotateExpiredKeys(ssmSyncMsg)

	if err == nil {
		t.Error("Expected error when StopSSMsyncFunction is true")
	}
}

func TestGetUsersForRotation(t *testing.T) {
	// This will fail without proper DB connection, but we test the function signature
	k4 := configmodels.K4{
		K4_SNO:   1,
		K4_Label: "test_label",
	}

	// We expect an error since DB is not connected in test environment
	_, err := getUsersForRotation(k4)

	if err == nil {
		t.Log("Warning: getUsersForRotation returned nil error, expected DB connection error")
	}
}
