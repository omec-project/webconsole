package ssmsync

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
)

// TODO: analise this implementation and add mutex to avoid race conditions

// var cfgChannel chan *configmodels.ConfigMessage

// Message structure for SSM synchronization
// List of actions: "SYNC_EXTERNAL_KEYS", "SYNC_USERS", "SYNC_OUR_KEYS", "HEALTH_CHECK" see below
// "KEY_ROTATION", "CHECK_KEY_LIFE"

var StopSSMsyncFunction bool = false

var (
	ErrorSyncChan     chan error = make(chan error, 10)
	ErrorRotationChan chan error = make(chan error, 10)
)

// Implementation of SSM synchronization logic
func SyncSsm(ssmSyncMsg chan *ssm.SsmSyncMessage, ssm ssm.SSM) {
	// A select statement to listen for messages or timers
	setSyncChanHandle(ssmSyncMsg)

	go ssm.SyncKeyListen(ssmSyncMsg)

	// Listen for rotation operations
	go ssm.KeyRotationListen(ssmSyncMsg)

	for {
		select {
		case err := <-ErrorSyncChan:
			logger.AppLog.Errorf("Detect a error in sync functions %s", err)
		case err := <-ErrorRotationChan:
			logger.AppLog.Errorf("Detect a error in rotation functions %s", err)
		}
	}
}
