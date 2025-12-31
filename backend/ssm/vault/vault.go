package vault

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	vaultsync "github.com/omec-project/webconsole/backend/ssm/vault_sync"
)

type VaultSSM struct{}

var Vault *VaultSSM = &VaultSSM{}

// Implement SSM interface methods for VaultSSM

// SyncKeyListen starts listening for key synchronization messages
func (v *VaultSSM) SyncKeyListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	logger.AppLog.Infof("Starting Vault key sync listener")
	vaultsync.SyncKeyListen(ssmSyncMsg)
}

// KeyRotationListen starts listening for key rotation events
func (v *VaultSSM) KeyRotationListen(ssmSyncMsg chan *ssm.SsmSyncMessage) {
	logger.AppLog.Infof("Starting Vault key rotation listener")
	vaultsync.KeyRotationListen(ssmSyncMsg)
}

// Login performs authentication to Vault based on configured method
// Tries mTLS, Kubernetes, and AppRole authentication in order
func (v *VaultSSM) Login() (string, error) {
	logger.AppLog.Infof("Attempting Vault login")

	token, err := apiclient.LoginVault()
	if err != nil {
		logger.WebUILog.Errorf("Error logging into Vault: %v", err)
		return "", err
	}

	logger.AppLog.Infof("Successfully logged into Vault")
	return token, nil
}

// HealthCheck performs a health check on the Vault connection
func (v *VaultSSM) HealthCheck() {
	logger.AppLog.Infof("Performing Vault health check")
	vaultsync.HealthCheckVault()
}

// InitDefault initializes Vault with default configuration
func (v *VaultSSM) InitDefault(ssmSyncMsg chan *ssm.SsmSyncMessage) error {
	logger.AppLog.Infof("Initializing Vault with default configuration")

	err := vaultsync.VaultSyncInitDefault(ssmSyncMsg)
	return err
}
