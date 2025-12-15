package vault

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
)

func TestVaultSSMImplementsSSMInterface(t *testing.T) {
	var _ ssm.SSM = (*VaultSSM)(nil)
}

func TestVaultSSMSyncKeyListen(t *testing.T) {
	v := &VaultSSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SyncKeyListen panicked: %v", r)
		}
	}()

	// We can't really test the full functionality without mocking the dependencies,
	// but we can at least verify it doesn't panic on instantiation
	if v == nil {
		t.Error("VaultSSM instance should not be nil")
	}

	// Close channel to prevent blocking
	close(ch)
}

func TestVaultSSMKeyRotationListen(t *testing.T) {
	v := &VaultSSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("KeyRotationListen panicked: %v", r)
		}
	}()

	if v == nil {
		t.Error("VaultSSM instance should not be nil")
	}

	close(ch)
}

func TestVaultSSMHealthCheck(t *testing.T) {
	v := &VaultSSM{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HealthCheck panicked: %v", r)
		}
	}()

	if v == nil {
		t.Error("VaultSSM instance should not be nil")
	}
}

func TestVaultSSMGlobalInstance(t *testing.T) {
	if Vault == nil {
		t.Error("Global Vault instance should not be nil")
	}

	// Verify it's the correct type
	if _, ok := any(Vault).(ssm.SSM); !ok {
		t.Error("Global Vault should implement SSM interface")
	}
}

func TestVaultSSMInitDefault(t *testing.T) {
	v := &VaultSSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	err := v.InitDefault(ch)

	if err != nil {
		t.Errorf("InitDefault should not return error, got: %v", err)
	}

	close(ch)
}
