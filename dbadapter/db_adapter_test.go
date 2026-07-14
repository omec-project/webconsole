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
