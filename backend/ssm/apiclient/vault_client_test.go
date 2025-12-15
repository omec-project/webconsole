package apiclient

import (
	"os"
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
)

func TestGetVaultClientInsecureNoMTLS(t *testing.T) {
	resetState()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{
		Vault: &factory.Vault{
			VaultUri:     "http://127.0.0.1:8200",
			TLS_Insecure: true,
		},
	}}

	client, err := GetVaultClient()
	if err != nil {
		t.Fatalf("unexpected error creating vault client: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil vault client")
	}

	// cached instance should be reused
	client2, err := GetVaultClient()
	if err != nil {
		t.Fatalf("unexpected error retrieving cached client: %v", err)
	}
	if client != client2 {
		t.Fatal("expected cached Vault client to be reused")
	}
}

func TestGetVaultClientMTLSFilesExist(t *testing.T) {
	resetState()

	crt, err := os.CreateTemp("", "vault-crt-*.pem")
	if err != nil {
		t.Fatalf("cannot create temp crt: %v", err)
	}
	defer os.Remove(crt.Name())

	key, err := os.CreateTemp("", "vault-key-*.pem")
	if err != nil {
		t.Fatalf("cannot create temp key: %v", err)
	}
	defer os.Remove(key.Name())

	ca, err := os.CreateTemp("", "vault-ca-*.pem")
	if err != nil {
		t.Fatalf("cannot create temp ca: %v", err)
	}
	defer os.Remove(ca.Name())

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{
		Vault: &factory.Vault{
			VaultUri: "http://127.0.0.1:8200",
			MTls: &factory.TLS2{
				Crt: crt.Name(),
				Key: key.Name(),
				Ca:  ca.Name(),
			},
		},
	}}

	client, err := GetVaultClient()
	if err != nil {
		t.Fatalf("expected success configuring mTLS: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil vault client")
	}
}
