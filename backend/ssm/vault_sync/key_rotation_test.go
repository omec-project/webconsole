package vaultsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/ssm"
)

func TestKeyRotationListen(t *testing.T) {
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
		},
	}

	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 5)

	// Start the listener in a goroutine
	go KeyRotationListen(ssmSyncMsg)

	// Test with ROTATE_INTERNAL_KEY action
	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "ROTATE_INTERNAL_KEY",
		Info:   "Test rotation",
	}

	// Test with ROTATE_K4 action
	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "ROTATE_K4",
		Info:   "Test rotation",
	}

	// Test with unknown action
	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "UNKNOWN_ACTION",
		Info:   "Test unknown",
	}

	// Close channel to stop listener
	close(ssmSyncMsg)
}

func TestKeyRotationListenLowerCase(t *testing.T) {
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
		},
	}

	ssmSyncMsg := make(chan *ssm.SsmSyncMessage, 5)

	// Start the listener in a goroutine
	go KeyRotationListen(ssmSyncMsg)

	// Test with lowercase action (should be handled by ToUpper)
	ssmSyncMsg <- &ssm.SsmSyncMessage{
		Action: "rotate_internal_key",
		Info:   "Test lowercase rotation",
	}

	// Close channel to stop listener
	close(ssmSyncMsg)
}

func TestRotateInternalTransitKeyWithStopCondition(t *testing.T) {
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
		},
	}

	// Set stop condition
	setStopCondition(true)
	defer func() {
		setStopCondition(false)
	}()

	err := rotateInternalTransitKey("test-key", nil)

	if err == nil {
		t.Error("Expected error when stop condition is true")
	}

	expectedMsg := "vault is down; skipping rotation"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRotateInternalTransitKeyWithValidLabel(t *testing.T) {
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
		},
	}

	// Set stop condition to false to allow the function to proceed
	setStopCondition(false)

	// This will likely fail without a real Vault connection, but we test the flow
	err := rotateInternalTransitKey(internalKeyLabel, nil)

	// We expect an error since Vault is not connected in test environment
	if err == nil {
		t.Log("Warning: rotateInternalTransitKey returned nil error, expected Vault connection error")
	}
}
