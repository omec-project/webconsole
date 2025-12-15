# Vault Integration for Webconsole

This document describes the Vault integration implemented for secure key management in the webconsole.

## Overview

The Vault integration provides secure storage and management of K4 encryption keys as an alternative or complement to the SSM (Secure Storage Module). Vault offers enterprise-grade secret management with multiple authentication methods and comprehensive audit logging.

## Architecture

The Vault integration follows the same pattern as the SSM integration:

```bash
configapi/handlers_k4.go  (API endpoints)
    ↓
configapi/ssm_api/vault_api.go  (API layer - StoreKey, UpdateKey, DeleteKey)
    ↓
configapi/ssm_api/vault_helpers.go  (Helper functions - Vault operations)
    ↓
backend/ssm/apiclient/vault_client.go  (Vault client)
    ↓
backend/ssm/apiclient/vault_login.go  (Authentication methods)
    ↓
Vault Server
```

### Key Components

1. **vault_api.go** - Implements the SSMAPI interface for Vault operations
2. **vault_helpers.go** - Helper functions for Vault KV operations (store, update, delete, get, list)
3. **vault_client.go** - Vault client initialization with TLS/mTLS support
4. **vault_login.go** - Multiple authentication methods (AppRole, Kubernetes, mTLS)
5. **vault.go** - Implements the SSM interface for Vault
6. **vault_sync/** - Synchronization and key rotation functions

## Authentication Methods

The integration supports three authentication methods, tried in this order:

### 1. mTLS (Mutual TLS) - Recommended for Production

Uses client certificates for authentication.

```yaml
vault:
  vault-uri: "https://vault.example.com:8200"
  allow-vault: true
  cert-role: "webconsole-cert-role"
  m-tls:
    crt: "/path/to/client-cert.crt"
    key: "/path/to/client-key.key"
    ca: "/path/to/ca-cert.crt"
```

**Setup in Vault:**

```bash
# Enable cert auth method
vault auth enable cert

# Configure certificate role
vault write auth/cert/certs/webconsole-cert-role \
    certificate=@ca.crt \
    allowed_common_names=webconsole \
    token_ttl=1h
```

### 2. Kubernetes Auth

Uses Kubernetes service account tokens for authentication.

```yaml
vault:
  vault-uri: "http://vault.vault.svc.cluster.local:8200"
  allow-vault: true
  k8s-role: "webconsole-role"
  k8s-jwt-path: "/var/run/secrets/kubernetes.io/serviceaccount/token"
```

**Setup in Vault:**

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc:443" \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt

# Create role
vault write auth/kubernetes/role/webconsole-role \
    bound_service_account_names=webconsole \
    bound_service_account_namespaces=default \
    policies=webconsole-policy \
    ttl=1h
```

### 3. AppRole Auth

Uses role ID and secret ID for authentication.

```yaml
vault:
  vault-uri: "https://vault.example.com:8200"
  allow-vault: true
  role-id: "your-role-id"
  secret-id: "your-secret-id"
```

**Setup in Vault:**

```bash
# Enable AppRole auth
vault auth enable approle

# Create role
vault write auth/approle/role/webconsole \
    secret_id_ttl=24h \
    token_ttl=1h \
    token_max_ttl=4h \
    policies=webconsole-policy

# Get role ID
vault read auth/approle/role/webconsole/role-id

# Generate secret ID
vault write -f auth/approle/role/webconsole/secret-id
```

## Vault Policy

Create a policy for the webconsole:

```hcl
# webconsole-policy.hcl
path "secret/data/k4keys/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "secret/metadata/k4keys/*" {
  capabilities = ["list", "read", "delete"]
}

path "sys/health" {
  capabilities = ["read"]
}
```

Apply the policy:

```bash
vault policy write webconsole-policy webconsole-policy.hcl
```

## Configuration

Add Vault configuration to your `webuiConfig.yml`:

```yaml
configuration:
  vault:
    vault-uri: "https://vault.example.com:8200"
    allow-vault: true
    tls-insecure: false  # Set to true only for development
    
    # Choose ONE authentication method:
    
    # Option 1: mTLS (recommended for production)
    cert-role: "webconsole-cert-role"
    m-tls:
      crt: "/etc/webconsole/certs/client.crt"
      key: "/etc/webconsole/certs/client.key"
      ca: "/etc/webconsole/certs/ca.crt"
    
    # Option 2: Kubernetes (for K8s deployments)
    k8s-role: "webconsole-role"
    k8s-jwt-path: "/var/run/secrets/kubernetes.io/serviceaccount/token"
    
    # Option 3: AppRole (for standalone deployments)
    role-id: "${VAULT_ROLE_ID}"  # Use environment variables
    secret-id: "${VAULT_SECRET_ID}"
```

## API Operations

### Store Key

When storing a K4 key, if Vault is enabled, the key is stored in:

- **Path:** `secret/data/k4keys/{key_label}-{key_id}`
- **Data:**
  - `key_label`: The label of the key (e.g., K4_AES256)
  - `key_value`: The hex-encoded key value
  - `key_type`: The type of key (e.g., AES256)
  - `key_id`: The sequence number

### Update Key

Updates an existing key at the same path.

### Delete Key

Deletes the key from Vault.

### Get Key

Retrieves a key by label and ID.

## Key Synchronization and Rotation

The Vault integration includes:

- **Health Checks:** Periodic checks every 30 seconds to ensure Vault is available
- **Key Sync:** Synchronizes keys between MongoDB and Vault every 5 minutes
- **Key Rotation:** Automatic rotation of keys older than 90 days
- **Daily Health Reports:** Daily checks on key age and expiration warnings

## Error Handling

The integration includes comprehensive error handling:

- Connection failures set a stop condition to prevent repeated failed operations
- All operations log errors with context
- Failures are returned to the API layer with appropriate HTTP status codes

## Testing

### Local Development with Vault

1. Start Vault in dev mode:

```bash
vault server -dev -dev-root-token-id="root"
```

2. Configure environment:

```bash
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'
```

3. Enable KV v2 secrets engine:

```bash
vault secrets enable -path=secret kv-v2
```

4. Update configuration:

```yaml
vault:
  vault-uri: "http://127.0.0.1:8200"
  allow-vault: true
  tls-insecure: true
  role-id: "test-role"
  secret-id: "test-secret"
```

## Troubleshooting

### Authentication Fails

Check logs for authentication errors:

```bash
grep "Vault login" /var/log/webconsole.log
```

Verify Vault is accessible:

```bash
curl -k https://vault.example.com:8200/v1/sys/health
```

### Key Storage Fails

Verify policy permissions:

```bash
vault token capabilities secret/data/k4keys/test
```

Check Vault audit logs:

```bash
vault audit enable file file_path=/var/log/vault/audit.log
```

### TLS Certificate Issues

Verify certificates:

```bash
openssl verify -CAfile ca.crt client.crt
openssl x509 -in client.crt -text -noout
```

## Security Best Practices

1. **Never use `tls-insecure: true` in production**
2. **Store sensitive credentials in environment variables or Kubernetes secrets**
3. **Use short-lived tokens with automatic renewal**
4. **Enable Vault audit logging**
5. **Implement proper certificate rotation**
6. **Use namespaces in multi-tenant environments**
7. **Monitor Vault health and key access patterns**
8. **Implement proper backup and disaster recovery for Vault**

## Avoiding Circular Import Issues

The implementation carefully avoids circular imports by:

1. **Separation of Concerns:**
   - `backend/ssm/vault/` - Implements SSM interface
   - `configapi/ssm_api/` - Implements SSMAPI interface
   - `backend/ssm/apiclient/` - Provides Vault client and authentication

2. **Dependency Direction:**
   - API handlers depend on `ssm_api`
   - `ssm_api` depends on `apiclient`
   - `vault_sync` can depend on `configapi` for database operations
   - `configapi` does NOT depend on `backend/ssm/vault` or `vault_sync`

3. **Interface-Based Design:**
   - Both SSM and SSMAPI use interfaces
   - Implementations are separated by package boundaries

## Related Files

- Configuration: [config/vault-config-sample.yml](../config/vault-config-sample.yml)
- API Handlers: [configapi/handlers_k4.go](../configapi/handlers_k4.go)
- Vault API: [configapi/ssm_api/vault_api.go](../configapi/ssm_api/vault_api.go)
- Vault Helpers: [configapi/ssm_api/vault_helpers.go](../configapi/ssm_api/vault_helpers.go)
- Vault Client: [backend/ssm/apiclient/vault_client.go](backend/ssm/apiclient/vault_client.go)
- Authentication: [backend/ssm/apiclient/vault_login.go](backend/ssm/apiclient/vault_login.go)
- Sync Functions: [backend/ssm/vault_sync/](backend/ssm/vault_sync/)
