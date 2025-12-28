package vaultsync

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/vault/api"
	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	ssmsync "github.com/omec-project/webconsole/backend/ssm/ssm_sync"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
	"golang.org/x/sync/errgroup"
)

var LatestKeyVersion int
var AuthSubsDatasMap = make(map[string]configmodels.SubsData)

// SyncUsers synchronizes user data encryption using Vault transit engine
func SyncUsers() {
	SyncUserMutex.Lock()
	defer SyncUserMutex.Unlock()

	coreVaultUserSync()
}

func coreVaultUserSync() {
	if readStopCondition() {
		logger.AppLog.Warn("Vault is down; skipping user sync")
		return
	}

	subsDatas, err := ssmsync.GetAllSubscriberData()
	if err != nil || len(subsDatas) == 0 {
		logger.AppLog.Error("Failed to get subscribers datas ")
	}

	for _, subData := range subsDatas {
		AuthSubsDatasMap[subData.UeId] = subData
	}

	logger.AppLog.Infof("Len for authSubsDataMap: %d", len(AuthSubsDatasMap))

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(int(factory.WebUIConfig.Configuration.Mongodb.ConcurrencyOps))
	for _, subsData := range subsDatas {
		logger.AppLog.Infof("Synchronizing user: %s", subsData.UeId)
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Check if user has no encryption assigned
			if subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm == 0 ||
				subsData.AuthenticationSubscription.K4_SNO == 0 || subsData.AuthenticationSubscription.PermanentKey.EncryptionKey == "" {
				logger.AppLog.Warnf("User %s has no encryption key assigned, encrypting with Vault transit", subsData.UeId)
				encryptUserDataVaultTransit(subsData, subsData.UeId)
			} else if subsData.AuthenticationSubscription.K4_SNO != 0 {
				// User has encryption, check if we need to rewrap (key rotation)
				logger.AppLog.Debugf("K4_SNO: %d   EncryptionAlgorithm: %d", subsData.AuthenticationSubscription.K4_SNO, subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm)
				logger.AppLog.Infof("User %s has existing encryption, checking for rewrap", subsData.UeId)
				rewrapUserDataVaultTransit(subsData, subsData.UeId)
			}
			return nil
		})
	}
	// Wait for all goroutines to finish and log any errors
	if err := g.Wait(); err != nil {
		logger.AppLog.Errorf("User synchronization completed with errors: %v", err)
	}
}

// getTransitKeysEncryptPath returns the transit keys encrypt path from configuration
func getTransitKeysEncryptPath() string {
	if factory.WebUIConfig != nil && factory.WebUIConfig.Configuration != nil && factory.WebUIConfig.Configuration.Vault != nil {
		if path := factory.WebUIConfig.Configuration.Vault.TransitKeysEncryptPath; path != "" {
			return path
		}
	}
	return "transit/encrypt"
}

// encryptUserDataVaultTransit encrypts user permanent key using Vault transit engine
func encryptUserDataVaultTransit(subsData configmodels.SubsData, ueId string) {
	if readStopCondition() {
		logger.AppLog.Warn("Vault is down; skipping user encryption")
		return
	}

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Failed to get Vault client: %v", err)
		return
	}

	if apiclient.VaultAuthToken == "" {
		if _, err := apiclient.LoginVault(); err != nil {
			logger.AppLog.Errorf("Failed to authenticate to Vault: %v", err)
			setStopCondition(true)
			return
		}
	}

	// Build AAD (Additional Authenticated Data) for context
	aad := fmt.Sprintf("%s-%d-%d", subsData.UeId, subsData.AuthenticationSubscription.K4_SNO, subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm)
	aadBytes := []byte(aad)

	// Encode plaintext to base64 for Vault
	plaintext := subsData.AuthenticationSubscription.PermanentKey.PermanentKeyValue
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(plaintext))

	// Prepare encrypt request for Vault transit
	encryptPath := fmt.Sprintf("%s/%s", getTransitKeysEncryptPath(), internalKeyLabel)
	encryptData := map[string]any{
		"plaintext": plaintextB64,
		"context":   base64.StdEncoding.EncodeToString(aadBytes), // AAD as context
	}

	secret, err := client.Logical().WriteWithContext(context.Background(), encryptPath, encryptData)
	if err != nil {
		logger.AppLog.Errorf("Failed to encrypt user data via Vault transit: %v", err)
		return
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		logger.AppLog.Errorf("No ciphertext returned from Vault transit encryption")
		return
	}

	ciphertext := secret.Data["ciphertext"].(string)

	// Update subscriber authentication data
	newSubAuthData := subsData.AuthenticationSubscription
	newSubAuthData.PermanentKey.PermanentKeyValue = ciphertext
	newSubAuthData.PermanentKey.EncryptionAlgorithm = ssm_constants.ALGORITHM_AES256_OurUsers // Mark as encrypted with Vault transit
	newSubAuthData.K4_SNO = 1                                                                 // Internal key ID (transit key)
	newSubAuthData.PermanentKey.Aad = hex.EncodeToString(aadBytes)
	newSubAuthData.PermanentKey.EncryptionKey = fmt.Sprintf("%s-%d", ssm_constants.LABEL_ENCRYPTION_KEY_AES256, 1)

	// Store updated data in MongoDB
	err = configapi.SubscriberAuthenticationDataUpdate(ueId, &newSubAuthData)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber %s: %v", ueId, err)
		return
	}
	logger.WebUILog.Infof("Subscriber %s encrypted and updated successfully with Vault transit", ueId)
}

// rewrapUserDataVaultTransit performs rewrapping if the transit key was rotated
func rewrapUserDataVaultTransit(subsData configmodels.SubsData, ueId string) {
	if readStopCondition() {
		logger.AppLog.Warn("Vault is down; skipping rewrap")
		return
	}

	client, err := apiclient.GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Failed to get Vault client: %v", err)
		return
	}

	if apiclient.VaultAuthToken == "" {
		if _, err := apiclient.LoginVault(); err != nil {
			logger.AppLog.Errorf("Failed to authenticate to Vault: %v", err)
			setStopCondition(true)
			return
		}
	}

	// Get current ciphertext from user data
	currentCiphertext := subsData.AuthenticationSubscription.PermanentKey.PermanentKeyValue

	// Extract version from ciphertext (format: vault:v1:...)
	ciphertextVersion, err := extractVersionFromCiphertext(currentCiphertext)
	if err != nil {
		logger.AppLog.Warnf("Failed to extract version from ciphertext for user %s: %v", ueId, err)
		return
	}

	// Get latest key version from Vault
	latestVersion, err := getLatestTransitKeyVersion(client, internalKeyLabel, "opt1")
	if err != nil {
		logger.AppLog.Errorf("Failed to get latest key version for user %s: %v", ueId, err)
		return
	}

	// Only rewrap if ciphertext version is older than latest version
	if ciphertextVersion >= latestVersion {
		logger.AppLog.Debugf("User %s ciphertext is already at version %d (latest: %d), no rewrap needed",
			ueId, ciphertextVersion, latestVersion)
		return
	}

	logger.AppLog.Infof("User %s ciphertext version %d is older than latest %d, performing rewrap",
		ueId, ciphertextVersion, latestVersion)

	// Rebuild AAD context
	aad := subsData.AuthenticationSubscription.PermanentKey.Aad
	var aadBytes []byte
	if aad != "" {
		aadBytes, _ = hex.DecodeString(aad)
	} else {
		// Fallback: rebuild AAD
		aadStr := fmt.Sprintf("%s-%d-%d", subsData.UeId, subsData.AuthenticationSubscription.K4_SNO, subsData.AuthenticationSubscription.PermanentKey.EncryptionAlgorithm)
		aadBytes = []byte(aadStr)
	}

	// Perform rewrap operation
	rewrapPath := fmt.Sprintf(getTransitKeyRewrapFormat(), internalKeyLabel)
	rewrapData := map[string]any{
		"ciphertext": currentCiphertext,
	}

	// Add context if AAD exists
	if len(aadBytes) > 0 {
		rewrapData["context"] = base64.StdEncoding.EncodeToString(aadBytes)
	}

	secret, err := client.Logical().WriteWithContext(context.Background(), rewrapPath, rewrapData)
	if err != nil {
		logger.AppLog.Errorf("Rewrap failed for user %s: %v", ueId, err)
		return
	}

	if secret == nil || secret.Data["ciphertext"] == nil {
		logger.AppLog.Errorf("No ciphertext returned from rewrap for user %s", ueId)
		return
	}

	newCiphertext := secret.Data["ciphertext"].(string)

	// Update subscriber authentication data with rewrapped ciphertext
	newSubAuthData := subsData.AuthenticationSubscription
	newSubAuthData.PermanentKey.PermanentKeyValue = newCiphertext

	// Store updated data in MongoDB
	err = configapi.SubscriberAuthenticationDataUpdate(ueId, &newSubAuthData)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber %s after rewrap: %v", ueId, err)
		return
	}
	logger.WebUILog.Infof("Subscriber %s rewrapped successfully from version %d to %d",
		ueId, ciphertextVersion, latestVersion)
}

// extractVersionFromCiphertext extracts the version number from a Vault ciphertext
// Ciphertext format: vault:v1:base64data or vault:v2:base64data
func extractVersionFromCiphertext(ciphertext string) (int, error) {
	// Check if it starts with "vault:"
	if !strings.HasPrefix(ciphertext, "vault:") {
		return 0, fmt.Errorf("invalid ciphertext format: does not start with 'vault:'")
	}

	// Split by colon to get parts: ["vault", "v1", "base64data"]
	parts := strings.SplitN(ciphertext, ":", 3)
	if len(parts) < 3 {
		return 0, fmt.Errorf("invalid ciphertext format: expected at least 3 parts")
	}

	// Extract version from second part (e.g., "v1" -> 1)
	versionStr := parts[1]
	if !strings.HasPrefix(versionStr, "v") {
		return 0, fmt.Errorf("invalid version format: does not start with 'v'")
	}

	// Parse the numeric part
	version, err := strconv.Atoi(versionStr[1:])
	if err != nil {
		return 0, fmt.Errorf("failed to parse version number: %w", err)
	}

	return version, nil
}

// getLatestTransitKeyVersion retrieves the latest version number of a transit key from Vault
func getLatestTransitKeyVersion(client *api.Client, keyName, opt string) (int, error) {
	if LatestKeyVersion != 0 && opt == "opt1" {
		return LatestKeyVersion, nil
	}
	// Read key information from Vault
	keyPath := fmt.Sprintf(getTransitKeyCreateFormat(), keyName)
	secret, err := client.Logical().Read(keyPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read key info: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return 0, fmt.Errorf("no data returned for key %s", keyName)
	}

	// Get latest_version field
	latestVersionRaw, ok := secret.Data["latest_version"]
	if !ok {
		return 0, fmt.Errorf("latest_version field not found in key data")
	}

	// Convert to int (Vault returns it as json.Number or int)
	var latestVersion int
	switch v := latestVersionRaw.(type) {
	case json.Number:
		// Handle json.Number type
		vInt, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("failed to convert json.Number to int: %w", err)
		}
		latestVersion = int(vInt)
	case int:
		latestVersion = v
	case float64:
		latestVersion = int(v)
	case int64:
		latestVersion = int(v)
	default:
		return 0, fmt.Errorf("unexpected type for latest_version: %T", latestVersionRaw)
	}

	LatestKeyVersion = latestVersion
	return latestVersion, nil
}
