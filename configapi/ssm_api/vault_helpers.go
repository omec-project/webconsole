package ssmapi

import (
	"context"
	"fmt"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
)

const (
	internalKeyLabel = "aes256-gcm"
)

// getVaultKeyPath returns the base KV path for keys from config with fallback
func getVaultKeyPath() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if p := factory.WebUIConfig.Configuration.Vault.KeyKVPath; p != "" {
			return p
		}
	}
	return "secret/data/k4keys"
}

// getTransitKeyCreateFormat returns the transit key create format from configuration
func getTransitPath() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if format := factory.WebUIConfig.Configuration.Vault.TransitKeysListPath; format != "" {
			return format
		}
	}
	return "transit/keys"
}

// StoreKeyVault stores a key in Vault's KV secrets engine
func StoreKeyVault(keyLabel, keyValue, keyType string, keyID int32) error {
	logger.AppLog.Debugf("Storing key in Vault - label: %s, id: %d, type: %s", keyLabel, keyID, keyType)

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return fmt.Errorf("error getting Vault client: %w", err)
	}

	// Build the secret path using label and ID
	secretPath := fmt.Sprintf("%s/%s-%d", getVaultKeyPath(), keyLabel, keyID)

	// Prepare the data to store
	data := map[string]any{
		"data": map[string]any{
			"key_label": keyLabel,
			"key_value": keyValue,
			"key_type":  keyType,
			"key_id":    keyID,
		},
	}

	// Write the secret to Vault
	_, err = client.Logical().WriteWithContext(context.Background(), secretPath, data)
	if err != nil {
		logger.AppLog.Errorf("Error writing key to Vault: %v", err)
		return fmt.Errorf("error writing key to Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully stored key in Vault at path: %s", secretPath)
	return nil
}

// UpdateKeyVault updates an existing key in Vault
func UpdateKeyVault(keyLabel, keyValue, keyType string, keyID int32) error {
	logger.AppLog.Debugf("Updating key in Vault - label: %s, id: %d, type: %s", keyLabel, keyID, keyType)

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return fmt.Errorf("error getting Vault client: %w", err)
	}

	// Build the secret path using label and ID
	secretPath := fmt.Sprintf("%s/%s-%d", getVaultKeyPath(), keyLabel, keyID)

	// Prepare the data to update
	data := map[string]any{
		"data": map[string]any{
			"key_label": keyLabel,
			"key_value": keyValue,
			"key_type":  keyType,
			"key_id":    keyID,
		},
	}

	// Write the secret to Vault (updates existing or creates new)
	_, err = client.Logical().WriteWithContext(context.Background(), secretPath, data)
	if err != nil {
		logger.AppLog.Errorf("Error updating key in Vault: %v", err)
		return fmt.Errorf("error updating key in Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully updated key in Vault at path: %s", secretPath)
	return nil
}

// DeleteKeyVault deletes a key from Vault
func DeleteKeyVault(keyLabel string, keyID int32) error {
	logger.AppLog.Debugf("Deleting key from Vault - label: %s, id: %d", keyLabel, keyID)

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return fmt.Errorf("error getting Vault client: %w", err)
	}

	// Build the secret path using label and ID
	secretPath := fmt.Sprintf("%s/%s-%d", getVaultKeyPath(), keyLabel, keyID)

	if keyLabel == ssm_constants.LABEL_ENCRYPTION_KEY_AES256 {
		logger.AppLog.Info("delete protected internal encryption key")
		secretPath = fmt.Sprintf("%s/%s", getTransitPath(), internalKeyLabel)
	}

	// Delete the secret from Vault
	_, err = client.Logical().DeleteWithContext(context.Background(), secretPath)
	if err != nil {
		logger.AppLog.Errorf("Error deleting key from Vault: %v", err)
		return fmt.Errorf("error deleting key from Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully deleted key from Vault at path: %s", secretPath)
	return nil
}

// GetKeyVault retrieves a key from Vault
func GetKeyVault(keyLabel string, keyID int32) (map[string]any, error) {
	logger.AppLog.Debugf("Retrieving key from Vault - label: %s, id: %d", keyLabel, keyID)

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return nil, fmt.Errorf("error getting Vault client: %w", err)
	}

	// Build the secret path using label and ID
	secretPath := fmt.Sprintf("%s/%s-%d", getVaultKeyPath(), keyLabel, keyID)

	// Read the secret from Vault
	secret, err := client.Logical().ReadWithContext(context.Background(), secretPath)
	if err != nil {
		logger.AppLog.Errorf("Error reading key from Vault: %v", err)
		return nil, fmt.Errorf("error reading key from Vault: %w", err)
	}

	if secret == nil {
		logger.AppLog.Warnf("Key not found in Vault at path: %s", secretPath)
		return nil, fmt.Errorf("key not found in Vault")
	}

	// Extract the data field from the secret
	data, ok := secret.Data["data"].(map[string]any)
	if !ok {
		logger.AppLog.Errorf("Invalid data format in Vault secret")
		return nil, fmt.Errorf("invalid data format in Vault secret")
	}

	logger.AppLog.Infof("Successfully retrieved key from Vault at path: %s", secretPath)
	return data, nil
}

// ListKeysVault lists all keys stored in Vault
func ListKeysVault() ([]string, error) {
	logger.AppLog.Debugf("Listing keys from Vault")

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return nil, fmt.Errorf("error getting Vault client: %w", err)
	}

	// List secrets at the key path
	secret, err := client.Logical().ListWithContext(context.Background(), getVaultKeyPath())
	if err != nil {
		logger.AppLog.Errorf("Error listing keys from Vault: %v", err)
		return nil, fmt.Errorf("error listing keys from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		logger.AppLog.Infof("No keys found in Vault")
		return []string{}, nil
	}

	// Extract the list of keys
	keys, ok := secret.Data["keys"].([]any)
	if !ok {
		logger.AppLog.Errorf("Invalid keys format in Vault list response")
		return nil, fmt.Errorf("invalid keys format in Vault list response")
	}

	// Convert to string slice
	keyList := make([]string, len(keys))
	for i, key := range keys {
		keyList[i] = key.(string)
	}

	logger.AppLog.Infof("Successfully listed %d keys from Vault", len(keyList))
	return keyList, nil
}
