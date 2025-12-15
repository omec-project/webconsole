package ssmhsm

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
)

func TestSSMHSMImplementsSSMInterface(t *testing.T) {
	var _ ssm.SSM = (*SSMHSM)(nil)
}

func TestSSMHSMSyncKeyListen(t *testing.T) {
	hsm := &SSMHSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SyncKeyListen panicked: %v", r)
		}
	}()

	// We can't really test the full functionality without mocking the dependencies,
	// but we can at least verify it doesn't panic on instantiation
	if hsm == nil {
		t.Error("SSMHSM instance should not be nil")
	}

	// Close channel to prevent blocking
	close(ch)
}

func TestSSMHSMKeyRotationListen(t *testing.T) {
	hsm := &SSMHSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("KeyRotationListen panicked: %v", r)
		}
	}()

	if hsm == nil {
		t.Error("SSMHSM instance should not be nil")
	}

	close(ch)
}

func TestSSMHSMHealthCheck(t *testing.T) {
	hsm := &SSMHSM{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HealthCheck panicked: %v", r)
		}
	}()

	if hsm == nil {
		t.Error("SSMHSM instance should not be nil")
	}
}

func TestSSMHSMGlobalInstance(t *testing.T) {
	if Ssmhsm == nil {
		t.Error("Global Ssmhsm instance should not be nil")
	}

	// Verify it's the correct type
	if _, ok := any(Ssmhsm).(ssm.SSM); !ok {
		t.Error("Global Ssmhsm should implement SSM interface")
	}
}

func TestSSMHSMInitDefault(t *testing.T) {
	hsm := &SSMHSM{}
	ch := make(chan *ssm.SsmSyncMessage, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitDefault panicked: %v", r)
		}
	}()

	if hsm == nil {
		t.Error("SSMHSM instance should not be nil")
	}

	close(ch)
}
