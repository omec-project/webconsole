package ssmapi

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
)

type VAULT_API struct{}

var Vault_api *VAULT_API = &VAULT_API{}

// StoreKey stores a K4 key in Vault
func (v *VAULT_API) StoreKey(k4Data *configmodels.K4) error {
	logger.AppLog.Infof("Storing key in Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	// Validate key label
	if k4Data.K4_Label == "" {
		logger.AppLog.Error("failed to store k4 key in Vault: label key is empty")
		return errors.New("failed to store k4 key in Vault: key label must be provided")
	}

	// Validate key type
	if k4Data.K4_Type == "" {
		logger.AppLog.Error("failed to store k4 key in Vault: key type is empty")
		return errors.New("failed to store k4 key in Vault: key type must be provided")
	}

	// Validate key value is hex
	if _, err := hex.DecodeString(k4Data.K4); err != nil {
		logger.AppLog.Errorf("failed to store k4 key in Vault: invalid hex string: %v", err)
		return errors.New("failed to store k4 key in Vault: key must be a valid hex string")
	}

	// Store the key in Vault
	err := StoreKeyVault(k4Data.K4_Label, k4Data.K4, k4Data.K4_Type, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to store k4 key in Vault: %+v", err)
		return fmt.Errorf("failed to store k4 key in Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully stored key in Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	// For Vault, we store the key in plaintext or encrypted form
	// The key value is kept as-is (no modification needed)
	return nil
}

// UpdateKey updates an existing K4 key in Vault
func (v *VAULT_API) UpdateKey(k4Data *configmodels.K4) error {
	logger.AppLog.Infof("Updating key in Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	// Validate key label
	if k4Data.K4_Label == "" {
		logger.AppLog.Error("failed to update k4 key in Vault: label key is empty")
		return errors.New("failed to update k4 key in Vault: key label must be provided")
	}

	// Validate key type
	if k4Data.K4_Type == "" {
		logger.AppLog.Error("failed to update k4 key in Vault: key type is empty")
		return errors.New("failed to update k4 key in Vault: key type must be provided")
	}

	// Validate key value is hex
	if _, err := hex.DecodeString(k4Data.K4); err != nil {
		logger.AppLog.Errorf("failed to update k4 key in Vault: invalid hex string: %v", err)
		return errors.New("failed to update k4 key in Vault: key must be a valid hex string")
	}

	// Update the key in Vault
	err := UpdateKeyVault(k4Data.K4_Label, k4Data.K4, k4Data.K4_Type, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to update k4 key in Vault: %+v", err)
		return fmt.Errorf("failed to update k4 key in Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully updated key in Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	return nil
}

// DeleteKey deletes a K4 key from Vault
func (v *VAULT_API) DeleteKey(k4Data *configmodels.K4) error {
	logger.AppLog.Infof("Deleting key from Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	// Validate key label
	if k4Data.K4_Label == "" {
		logger.AppLog.Error("failed to delete k4 key in Vault: label key is empty")
		return errors.New("failed to delete k4 key in Vault: key label must be provided")
	}

	// Delete the key from Vault
	err := DeleteKeyVault(k4Data.K4_Label, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to delete k4 key in Vault: %+v", err)
		return fmt.Errorf("failed to delete k4 key in Vault: %w", err)
	}

	logger.AppLog.Infof("Successfully deleted key from Vault: Label=%s, SNO=%d", k4Data.K4_Label, k4Data.K4_SNO)

	return nil
}

// IsValidKeyIdentifierVault validates if a key identifier is in the allowed list
func IsValidKeyIdentifierVault(keyLabel string, allowedIdentifiers []string) bool {
	if keyLabel == "" {
		return false
	}
	return slices.Contains(allowedIdentifiers, keyLabel)
}

// EncodeKeyToBase64 encodes a hex string key to base64 for Vault storage
func EncodeKeyToBase64(hexKey string) (string, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(keyBytes), nil
}

// DecodeKeyFromBase64 decodes a base64 key from Vault to hex string
func DecodeKeyFromBase64(base64Key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 key: %w", err)
	}
	return hex.EncodeToString(keyBytes), nil
}
