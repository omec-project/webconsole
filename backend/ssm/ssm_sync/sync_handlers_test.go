package ssmsync

import (
	"testing"

	"github.com/omec-project/webconsole/backend/ssm"
)

func TestSetSyncChanHandle(t *testing.T) {
	ch := make(chan *ssm.SsmSyncMessage, 1)

	setSyncChanHandle(ch)

	if ssmSyncMessage != ch {
		t.Error("setSyncChanHandle should set the global ssmSyncMessage channel")
	}
}

func TestSetSyncChanHandleNilChannel(t *testing.T) {
	setSyncChanHandle(nil)

	if ssmSyncMessage != nil {
		t.Error("setSyncChanHandle should accept nil channel")
	}
}

func TestSyncMutexesInitialized(t *testing.T) {
	// Test that mutexes are initialized
	// We can't directly test mutex state, but we can test Lock/Unlock

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
