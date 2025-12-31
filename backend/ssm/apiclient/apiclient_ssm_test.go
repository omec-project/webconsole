package apiclient

import (
	"context"
	"net/http"
	"testing"

	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/factory"
)

// helper to reset globals between tests
func resetState() {
	apiClient = nil
	ResetVaultClient()
	AuthContext = context.Background()
	CurrentJWT = ""
}

func TestSetAuthContext(t *testing.T) {
	resetState()
	token := "test-token"
	SetAuthContext(token)

	if CurrentJWT != token {
		t.Fatalf("expected CurrentJWT %s, got %s", token, CurrentJWT)
	}

	ctxVal := AuthContext.Value(ssm_models.ContextAccessToken)
	if ctxVal != token {
		t.Fatalf("expected context token %s, got %v", token, ctxVal)
	}
}

func TestGetHTTPClientInsecure(t *testing.T) {
	client := GetHTTPClient(true)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http.Transport, got %T", client.Transport)
	}

	tlsCfg := transport.TLSClientConfig
	if tlsCfg == nil || !tlsCfg.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify true, got %#v", tlsCfg)
	}
}

func TestGetHTTPClientSecure(t *testing.T) {
	client := GetHTTPClient(false)
	if client.Transport != nil {
		t.Fatalf("expected default transport when secure, got %T", client.Transport)
	}
}

func TestGetSSMAPIClientCaching(t *testing.T) {
	resetState()
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				SsmUri:       "https://ssm.example.com",
				TLS_Insecure: true,
			},
		},
	}

	first := GetSSMAPIClient()
	if first == nil {
		t.Fatal("expected non-nil SSM API client")
	}

	second := GetSSMAPIClient()
	if first != second {
		t.Fatal("expected cached SSM API client to be reused")
	}
}

func TestGetVaultClientCaching(t *testing.T) {
	resetState()
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			Vault: &factory.Vault{
				VaultUri:     "http://127.0.0.1:8200",
				TLS_Insecure: true,
			},
		},
	}

	first, err := GetVaultClient()
	if err != nil {
		t.Fatalf("unexpected error creating vault client: %v", err)
	}

	second, err := GetVaultClient()
	if err != nil {
		t.Fatalf("unexpected error retrieving cached vault client: %v", err)
	}

	if first != second {
		t.Fatal("expected cached Vault client to be reused")
	}
}

func TestGetVaultClientMissingCertFiles(t *testing.T) {
	resetState()
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			Vault: &factory.Vault{
				VaultUri:     "http://127.0.0.1:8200",
				TLS_Insecure: true,
				MTls: &factory.TLS2{
					Crt: "nonexistent.crt",
					Key: "nonexistent.key",
					Ca:  "nonexistent.ca",
				},
			},
		},
	}

	_, err := GetVaultClient()
	if err == nil {
		t.Fatal("expected error when certificate files are missing, got nil")
	}
}

func TestLoginVaultNoMethodsConfigured(t *testing.T) {
	resetState()
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			Vault: &factory.Vault{VaultUri: "http://127.0.0.1:8200"},
		},
	}

	if _, err := LoginVault(); err == nil {
		t.Fatal("expected authentication failure when no methods are configured")
	}
}
