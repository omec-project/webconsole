package apiclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
)

func TestLoginVaultAppRoleSuccess(t *testing.T) {
	resetState()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"auth":{"client_token":"tok-approle","accessor":"acc"}}`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{Vault: &factory.Vault{VaultUri: server.URL, TLS_Insecure: true}}}

	tok, err := LoginVaultAppRole("role", "secret")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tok != "tok-approle" {
		t.Fatalf("expected token tok-approle, got %s", tok)
	}
	if VaultAuthToken != "tok-approle" {
		t.Fatalf("expected VaultAuthToken cached, got %s", VaultAuthToken)
	}
}

func TestLoginVaultKubernetesSuccess(t *testing.T) {
	resetState()

	jwtFile, err := os.CreateTemp("", "jwt")
	if err != nil {
		t.Fatalf("cannot create temp jwt file: %v", err)
	}
	defer os.Remove(jwtFile.Name())
	if _, err = jwtFile.WriteString("dummy-jwt"); err != nil {
		t.Fatalf("Failed to write JWT file: %v", err)
	}
	jwtFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/kubernetes/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"auth":{"client_token":"tok-k8s","accessor":"acc"}}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{Vault: &factory.Vault{VaultUri: server.URL, TLS_Insecure: true}}}

	tok, err := LoginVaultKubernetes("role", jwtFile.Name())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tok != "tok-k8s" {
		t.Fatalf("expected token tok-k8s, got %s", tok)
	}
	if VaultAuthToken != "tok-k8s" {
		t.Fatalf("expected VaultAuthToken cached, got %s", VaultAuthToken)
	}
}

func TestLoginVaultMTLSSuccess(t *testing.T) {
	resetState()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/cert/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"auth":{"client_token":"tok-mtls","accessor":"acc"}}`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// No mTLS files needed for this logical call; we rely on transit Write
	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{Vault: &factory.Vault{VaultUri: server.URL, TLS_Insecure: true}}}

	tok, err := LoginVaultMTLS("", "")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tok != "tok-mtls" {
		t.Fatalf("expected token tok-mtls, got %s", tok)
	}
	if VaultAuthToken != "tok-mtls" {
		t.Fatalf("expected VaultAuthToken cached, got %s", VaultAuthToken)
	}
}

func TestLoginVaultPrefersK8s(t *testing.T) {
	resetState()

	jwtFile, err := os.CreateTemp("", "jwt")
	if err != nil {
		t.Fatalf("cannot create temp jwt file: %v", err)
	}
	defer os.Remove(jwtFile.Name())
	if _, err = jwtFile.WriteString("dummy-jwt"); err != nil {
		t.Fatalf("Failed to write JWT file: %v", err)
	}
	jwtFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/kubernetes/login":
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{"auth":{"client_token":"tok-k8s","accessor":"acc"}}`))
			if err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		case "/v1/auth/approle/login":
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(`{"errors":["should not hit approle"]}`))
			if err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{Vault: &factory.Vault{
		VaultUri:     server.URL,
		TLS_Insecure: true,
		K8sRole:      "role",
		K8sJWTPath:   jwtFile.Name(),
		RoleID:       "role-id",
		SecretID:     "secret-id",
	}}}

	tok, err := LoginVault()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tok != "tok-k8s" {
		t.Fatalf("expected token tok-k8s, got %s", tok)
	}
}
