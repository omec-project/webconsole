package apiclient

import (
	"context"
	"fmt"
	"os"

	auth "github.com/hashicorp/vault/api/auth/approle"
	k8sauth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
)

var VaultAuthToken string = ""

// LoginVaultAppRole performs AppRole authentication to Vault
// Returns the authentication token
func LoginVaultAppRole(roleID, secretID string) (string, error) {
	logger.AppLog.Infof("Attempting Vault login using AppRole authentication")

	client, err := GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return "", fmt.Errorf("error getting Vault client: %w", err)
	}

	// Set login options for AppRole authentication
	opts := []auth.LoginOption{}

	// Add custom mount path if configured
	config := factory.WebUIConfig.Configuration.Vault
	if config.AppRoleMountPath != "" {
		opts = append(opts, auth.WithMountPath(config.AppRoleMountPath))
		logger.AppLog.Infof("Using custom AppRole mount path: %s", config.AppRoleMountPath)
	}

	// Create AppRole auth method
	appRoleAuth, err := auth.NewAppRoleAuth(roleID, &auth.SecretID{
		FromString: secretID,
	}, opts...)
	if err != nil {
		logger.AppLog.Errorf("Error creating AppRole auth: %v", err)
		return "", fmt.Errorf("error creating AppRole auth: %w", err)
	}

	// Authenticate
	authInfo, err := client.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		logger.AppLog.Errorf("Error logging in with AppRole: %v", err)
		return "", fmt.Errorf("error logging in with AppRole: %w", err)
	}

	if authInfo == nil {
		logger.AppLog.Errorf("No auth info returned from Vault")
		return "", fmt.Errorf("no auth info returned from Vault")
	}

	// Set the token
	token := authInfo.Auth.ClientToken
	client.SetToken(token)
	VaultAuthToken = token

	logger.AppLog.Infof("Successfully authenticated to Vault using AppRole")
	logger.AppLog.Debugf("Token accessor: %s", authInfo.Auth.Accessor)

	return token, nil
}

// LoginVaultKubernetes performs Kubernetes authentication to Vault
// Returns the authentication token
func LoginVaultKubernetes(role, jwtPath string) (string, error) {
	logger.AppLog.Infof("Attempting Vault login using Kubernetes authentication")

	client, err := GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return "", fmt.Errorf("error getting Vault client: %w", err)
	}

	// If no JWT path provided, use default service account token path
	if jwtPath == "" {
		jwtPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		logger.AppLog.Debugf("Using default Kubernetes service account token path: %s", jwtPath)
	}

	// Read the JWT token
	jwt, err := os.ReadFile(jwtPath)
	if err != nil {
		logger.AppLog.Errorf("Error reading Kubernetes JWT token: %v", err)
		return "", fmt.Errorf("error reading Kubernetes JWT token: %w", err)
	}

	// Create Kubernetes auth method with optional custom mount path
	k8sOpts := []k8sauth.LoginOption{k8sauth.WithServiceAccountToken(string(jwt))}
	config := factory.WebUIConfig.Configuration.Vault
	if config.K8sMountPath != "" {
		k8sOpts = append(k8sOpts, k8sauth.WithMountPath(config.K8sMountPath))
		logger.AppLog.Infof("Using custom Kubernetes mount path: %s", config.K8sMountPath)
	}

	k8sAuth, err := k8sauth.NewKubernetesAuth(role, k8sOpts...)
	if err != nil {
		logger.AppLog.Errorf("Error creating Kubernetes auth: %v", err)
		return "", fmt.Errorf("error creating Kubernetes auth: %w", err)
	}

	// Authenticate
	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		logger.AppLog.Errorf("Error logging in with Kubernetes auth: %v", err)
		return "", fmt.Errorf("error logging in with Kubernetes auth: %w", err)
	}

	if authInfo == nil {
		logger.AppLog.Errorf("No auth info returned from Vault")
		return "", fmt.Errorf("no auth info returned from Vault")
	}

	// Set the token
	token := authInfo.Auth.ClientToken
	client.SetToken(token)
	VaultAuthToken = token

	logger.AppLog.Infof("Successfully authenticated to Vault using Kubernetes")
	logger.AppLog.Debugf("Token accessor: %s", authInfo.Auth.Accessor)

	return token, nil
}

// LoginVaultMTLS performs mTLS authentication to Vault
// The mTLS certificates are configured when creating the client
// This method validates the authentication
func LoginVaultMTLS(certPath, certRole string) (string, error) {
	logger.AppLog.Infof("Attempting Vault login using mTLS authentication")

	client, err := GetVaultClient()
	if err != nil {
		logger.AppLog.Errorf("Error getting Vault client: %v", err)
		return "", fmt.Errorf("error getting Vault client: %w", err)
	}

	// For mTLS (TLS Certificate auth), we need to login through the cert auth method
	// The certificates are already configured in the HTTP client
	data := map[string]any{}
	if certRole != "" {
		data["name"] = certRole
	}

	// Use custom mount path if configured
	config := factory.WebUIConfig.Configuration.Vault
	certMountPath := "auth/cert/login"
	if config.CertMountPath != "" {
		certMountPath = fmt.Sprintf("auth/%s/login", config.CertMountPath)
		logger.AppLog.Infof("Using custom Cert mount path: %s", config.CertMountPath)
	}

	// Authenticate using cert auth method
	secret, err := client.Logical().Write(certMountPath, data)
	if err != nil {
		logger.AppLog.Errorf("Error logging in with mTLS: %v", err)
		return "", fmt.Errorf("error logging in with mTLS: %w", err)
	}

	if secret == nil || secret.Auth == nil {
		logger.AppLog.Errorf("No auth info returned from Vault")
		return "", fmt.Errorf("no auth info returned from Vault")
	}

	// Set the token
	token := secret.Auth.ClientToken
	client.SetToken(token)
	VaultAuthToken = token

	logger.AppLog.Infof("Successfully authenticated to Vault using mTLS")
	logger.AppLog.Debugf("Token accessor: %s", secret.Auth.Accessor)

	return token, nil
}

// LoginVault performs Vault authentication based on configuration
// It tries authentication methods in the following order:
// 1. mTLS (if MTls config is present)
// 2. Kubernetes (if in a Kubernetes environment)
// 3. AppRole (if AppRole credentials are configured)
func LoginVault() (string, error) {
	config := factory.WebUIConfig.Configuration.Vault

	// Try mTLS first if configured
	if config.MTls != nil && config.MTls.Crt != "" && config.MTls.Key != "" {
		logger.AppLog.Infof("Attempting mTLS authentication")
		token, err := LoginVaultMTLS(config.MTls.Crt, config.CertRole)
		if err == nil {
			return token, nil
		}
		logger.AppLog.Warnf("mTLS authentication failed: %v, trying next method", err)
	}

	// Try Kubernetes authentication if in Kubernetes environment
	if config.K8sRole != "" {
		logger.AppLog.Infof("Attempting Kubernetes authentication")
		token, err := LoginVaultKubernetes(config.K8sRole, config.K8sJWTPath)
		if err == nil {
			return token, nil
		}
		logger.AppLog.Warnf("Kubernetes authentication failed: %v, trying next method", err)
	}

	// Try AppRole authentication
	if config.RoleID != "" && config.SecretID != "" {
		logger.AppLog.Infof("Attempting AppRole authentication")
		token, err := LoginVaultAppRole(config.RoleID, config.SecretID)
		if err == nil {
			return token, nil
		}
		logger.AppLog.Warnf("AppRole authentication failed: %v", err)
	}

	// If all methods fail
	logger.AppLog.Errorf("All Vault authentication methods failed")
	return "", fmt.Errorf("failed to authenticate to Vault: no valid authentication method succeeded")
}
