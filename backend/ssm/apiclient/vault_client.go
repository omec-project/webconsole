package apiclient

import (
	"fmt"
	"os"
	"sync"

	vault "github.com/hashicorp/vault/api"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
)

var vaultClient *vault.Client
var mutexVaultClient sync.Mutex

// GetVaultClient creates and returns a configured Vault API client
func GetVaultClient() (*vault.Client, error) {
	mutexVaultClient.Lock()
	defer mutexVaultClient.Unlock()
	if vaultClient != nil {
		logger.AppLog.Debugf("Returning existing Vault client")
		return vaultClient, nil
	}

	logger.AppLog.Infof("Creating new Vault client for URI: %s", factory.WebUIConfig.Configuration.Vault.VaultUri)

	config := vault.DefaultConfig()
	config.Address = factory.WebUIConfig.Configuration.Vault.VaultUri

	// Prepare TLS configuration
	tlsConfig := &vault.TLSConfig{
		Insecure: factory.WebUIConfig.Configuration.Vault.TLS_Insecure,
	}

	// Handle insecure TLS
	if factory.WebUIConfig.Configuration.Vault.TLS_Insecure {
		logger.AppLog.Warnf("TLS_Insecure enabled - skipping certificate verification")
	}

	// Configure mTLS if enabled
	if factory.WebUIConfig.Configuration.Vault.MTls != nil {
		logger.AppLog.Infof("Configuring mTLS for Vault client")

		// Verify certificate files exist
		logger.AppLog.Debugf("Loading client certificate from: %s", factory.WebUIConfig.Configuration.Vault.MTls.Crt)
		if _, err := os.Stat(factory.WebUIConfig.Configuration.Vault.MTls.Crt); err != nil {
			logger.AppLog.Errorf("Client certificate file not found: %v", err)
			return nil, fmt.Errorf("client certificate file not found: %w", err)
		}

		logger.AppLog.Debugf("Loading client key from: %s", factory.WebUIConfig.Configuration.Vault.MTls.Key)
		if _, err := os.Stat(factory.WebUIConfig.Configuration.Vault.MTls.Key); err != nil {
			logger.AppLog.Errorf("Client key file not found: %v", err)
			return nil, fmt.Errorf("client key file not found: %w", err)
		}

		logger.AppLog.Debugf("Loading CA certificate from: %s", factory.WebUIConfig.Configuration.Vault.MTls.Ca)
		if _, err := os.Stat(factory.WebUIConfig.Configuration.Vault.MTls.Ca); err != nil {
			logger.AppLog.Errorf("CA certificate file not found: %v", err)
			return nil, fmt.Errorf("CA certificate file not found: %w", err)
		}

		// Set certificate paths in Vault TLS config
		tlsConfig.ClientCert = factory.WebUIConfig.Configuration.Vault.MTls.Crt
		tlsConfig.ClientKey = factory.WebUIConfig.Configuration.Vault.MTls.Key
		tlsConfig.CACert = factory.WebUIConfig.Configuration.Vault.MTls.Ca

		logger.AppLog.Infof("mTLS configuration completed successfully")
	}

	// Apply TLS configuration to Vault client
	if err := config.ConfigureTLS(tlsConfig); err != nil {
		logger.AppLog.Errorf("Error configuring TLS for Vault client: %v", err)
		return nil, fmt.Errorf("error configuring TLS: %w", err)
	}

	// Create Vault client
	client, err := vault.NewClient(config)
	if err != nil {
		logger.AppLog.Errorf("Error creating Vault client: %v", err)
		return nil, fmt.Errorf("error creating Vault client: %w", err)
	}

	vaultClient = client
	logger.AppLog.Infof("Vault client created successfully")

	return vaultClient, nil
}

// ResetVaultClient resets the cached Vault client (useful for testing or re-authentication)
func ResetVaultClient() {
	vaultClient = nil
	logger.AppLog.Debugf("Vault client reset")
}
