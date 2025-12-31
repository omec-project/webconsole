package apiclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
)

func TestLoginSSMSuccess(t *testing.T) {
	resetState()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"token":"jwt123","message":"ok"}`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{
		SSM: &factory.SSM{SsmUri: server.URL, TLS_Insecure: true},
	}}

	tok, err := LoginSSM("svc", "pwd")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tok != "jwt123" {
		t.Fatalf("expected token jwt123, got %s", tok)
	}
	if CurrentJWT != "jwt123" {
		t.Fatalf("expected CurrentJWT set, got %s", CurrentJWT)
	}
}

func TestLoginSSMError(t *testing.T) {
	resetState()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"message":"fail"}`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	factory.WebUIConfig = &factory.Config{Configuration: &factory.Configuration{
		SSM: &factory.SSM{SsmUri: server.URL, TLS_Insecure: true},
	}}

	if _, err := LoginSSM("svc", "pwd"); err == nil {
		t.Fatal("expected error when backend returns 500")
	}
}
