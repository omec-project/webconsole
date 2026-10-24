// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package dbadapter

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type mockIndexDB struct {
	responses []indexResponse
	calls     int
}

type indexResponse struct {
	resp bool
	err  error
}

func (m *mockIndexDB) CreateIndex(collName string, keyField string) (bool, error) {
	if len(m.responses) == 0 {
		m.calls++
		return true, nil
	}

	idx := m.calls
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.calls++
	return m.responses[idx].resp, m.responses[idx].err
}

func retryableWriteException(message string) error {
	return mongo.WriteException{
		WriteConcernError: &mongo.WriteConcernError{
			Message: message,
		},
	}
}

func TestCreateIndexWithRetry(t *testing.T) {
	t.Run("succeeds after transient retryable error", func(t *testing.T) {
		client := &mockIndexDB{
			responses: []indexResponse{
				{resp: false, err: retryableWriteException("write concern error: (InterruptedAtShutdown) interrupted at shutdown")},
				{resp: true, err: nil},
			},
		}

		err := createIndexWithRetry(client, "upf", "hostname", 100*time.Millisecond, 5*time.Millisecond)
		if err != nil {
			t.Fatalf("expected retry to succeed, got %v", err)
		}
		if client.calls < 2 {
			t.Fatalf("expected at least 2 CreateIndex calls, got %d", client.calls)
		}
	})

	t.Run("fails fast on permanent error", func(t *testing.T) {
		client := &mockIndexDB{
			responses: []indexResponse{
				{resp: false, err: fmt.Errorf("duplicate key constraint mismatch")},
			},
		}

		err := createIndexWithRetry(client, "upf", "hostname", 100*time.Millisecond, 5*time.Millisecond)
		if err == nil {
			t.Fatal("expected permanent error")
		}
		if !strings.Contains(err.Error(), "duplicate key") {
			t.Fatalf("expected permanent error to be returned, got %v", err)
		}
		if client.calls != 1 {
			t.Fatalf("expected 1 CreateIndex call, got %d", client.calls)
		}
	})

	t.Run("times out on repeated transient error", func(t *testing.T) {
		client := &mockIndexDB{
			responses: []indexResponse{
				{resp: false, err: retryableWriteException("InterruptedAtShutdown")},
			},
		}

		err := createIndexWithRetry(client, "upf", "hostname", 30*time.Millisecond, 5*time.Millisecond)
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "timed out creating index") {
			t.Fatalf("expected timeout error, got %v", err)
		}
		if client.calls < 1 {
			t.Fatalf("expected CreateIndex to be called at least once")
		}
	})
}

// redactedSingleHost is how redactURL logs a one-host URL with credentials.
const redactedSingleHost = "mongodb://aether:xxxxx@mongodb-svc:27017/"

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "password replaced",
			in:   "mongodb://aether:s3cret@mongodb-svc:27017/?replicaSet=mongodb&authSource=admin",
			want: "mongodb://aether:xxxxx@mongodb-svc:27017/?replicaSet=mongodb&authSource=admin",
		},
		{
			name: "percent-encoded password replaced",
			in:   "mongodb://aether:s3cr%20et%2Bx@mongodb-svc:27017/",
			want: redactedSingleHost,
		},
		{
			name: "several hosts kept",
			in:   "mongodb://aether:s3cret@mongodb-0:27017,mongodb-1:27017/?replicaSet=rs0",
			want: "mongodb://aether:xxxxx@mongodb-0:27017,mongodb-1:27017/?replicaSet=rs0",
		},
		{
			name: "no credentials unchanged",
			in:   "mongodb://mongodb-headless:27017/?replicaSet=rs0",
			want: "mongodb://mongodb-headless:27017/?replicaSet=rs0",
		},
		{
			name: "SRV URL",
			in:   "mongodb+srv://aether:s3cret@cluster.example.com/aether?retryWrites=true",
			want: "mongodb+srv://aether:xxxxx@cluster.example.com/aether?retryWrites=true",
		},
		{
			name: "user without password unchanged",
			in:   "mongodb://aether@mongodb-svc:27017/",
			want: "mongodb://aether@mongodb-svc:27017/",
		},
		{
			name: "invalid escape in the password still replaced",
			in:   "mongodb://aether:s3%zzret@mongodb-svc:27017/",
			want: redactedSingleHost,
		},
		{
			name: "unescaped @ in the password replaced",
			in:   "mongodb://aether:s3@cret@mongodb-svc:27017/",
			want: redactedSingleHost,
		},
		{
			name: "unescaped ? in the password, not logged",
			in:   "mongodb://aether:s3?cret@mongodb-svc:27017/",
			want: urlNotLogged,
		},
		{
			name: "unescaped / in the password, not logged",
			in:   "mongodb://aether:s3/cret@mongodb-svc:27017/",
			want: urlNotLogged,
		},
		{
			name: "@ in the options, not logged",
			in:   "mongodb://mongodb-svc:27017/?appName=webui@site",
			want: urlNotLogged,
		},
		{
			name: "no scheme with credentials, not logged",
			in:   "aether:s3cret@mongodb-svc:27017",
			want: urlNotLogged,
		},
		{
			name: "password where the scheme should be, not logged",
			in:   "aether:s3cret://tail@mongodb-svc:27017/",
			want: urlNotLogged,
		},
		{
			name: "other scheme without credentials unchanged",
			in:   "http://mongodb-svc:27017/",
			want: "http://mongodb-svc:27017/",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := redactURL(tc.in)
			if got != tc.want {
				t.Errorf("redactURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
			for _, secret := range []string{"s3cret", "s3cr%20et", "s3%zzret", "s3@cret", "s3?cret", "s3/cret", "cret@"} {
				if strings.Contains(got, secret) {
					t.Errorf("redactURL(%q) = %q still contains the password", tc.in, got)
				}
			}
		})
	}
}
