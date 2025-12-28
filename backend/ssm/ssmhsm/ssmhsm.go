package ssmhsm

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	ssmsync "github.com/omec-project/webconsole/backend/ssm/ssm_sync"
	"github.com/omec-project/webconsole/backend/utils"
)

type SSMHSM struct{}

var Ssmhsm *SSMHSM = &SSMHSM{}

// Implement SSM interface methods for SSMHSM
func (hsm *SSMHSM) SyncKeyListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	// Implementation for syncing keys with HSM
	ssmsync.SyncKeyListen(ssmSyncMsg)
}

func (hsm *SSMHSM) KeyRotationListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	// Implementation for key rotation with HSM
	ssmsync.KeyRotationListen(ssmSyncMsg)
}

func (hsm *SSMHSM) Login() (string, error) {
	// Implementation for HSM login
	serviceId, password, err := utils.GetUserLogin()
	if err != nil {
		logger.WebUILog.Errorf("Error getting SSM login credentials: %v", err)
		return "", err
	}
	token, err := apiclient.LoginSSM(serviceId, password)
	if err != nil {
		logger.WebUILog.Errorf("Error logging into SSM: %v", err)
		return "", err
	}

	return token, nil
}

func (hsm *SSMHSM) HealthCheck() {
	// Implementation for HSM health check
	ssmsync.HealthCheckSSM()
}

func (hsm *SSMHSM) InitDefault(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	ssmsync.SsmSyncInitDefault(ssmSyncMsg)
	return nil
}
