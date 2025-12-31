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

	// Write minimal valid PEM content
	crtContent := `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyQVjOWIYBZJCfqJHCBa2
JjCCQZYzLJHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKH
kP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5J
lKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHv
NqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8
l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5
JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKH
kP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5J
lKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHv
NqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8
l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5JtKHkP5JlKHvNqP8l5m5
-----END CERTIFICATE-----`

	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDJBWM5YhgFkkJ+
okcIFrYmMIJBljMske82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yX
mbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm
0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ
/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmU
oe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82
o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yX
mbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm
0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm0oeQ/kmUoe82o/yXmbkm
-----END PRIVATE KEY-----`

	if _, err = crt.WriteString(crtContent); err != nil {
		t.Fatalf("cannot write to temp crt file: %v", err)
	}
	if err = crt.Close(); err != nil {
		t.Fatalf("cannot close temp crt file: %v", err)
	}

	if _, err = key.WriteString(keyContent); err != nil {
		t.Fatalf("cannot write to temp key file: %v", err)
	}
	if err = key.Close(); err != nil {
		t.Fatalf("cannot close temp key file: %v", err)
	}

	if _, err = ca.WriteString(crtContent); err != nil {
		t.Fatalf("cannot write to temp ca file: %v", err)
	}
	if err = ca.Close(); err != nil {
		t.Fatalf("cannot close temp ca file: %v", err)
	}

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

	// Since the certificates are dummy/invalid, we expect an error
	client, err := GetVaultClient()
	if err == nil {
		t.Fatal("expected error with invalid certificates, but got success")
	}
	if client != nil {
		t.Fatal("expected nil client when certificate configuration fails")
	}
}
