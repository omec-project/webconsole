// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package dbadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/omec-project/util/mongoapi"
)

var testIndexSpec = mongoapi.IndexSpec{
	Name: "amDataByUeIdAndServingPlmnId",
	Keys: mongoapi.AscendingKeys("ueId", "servingPlmnId"),
}

// mockIndexEnsurer answers EnsureIndex from errs, one per call, and succeeds
// once errs is exhausted. With hangFirst, the first call blocks until its
// context ends.
type mockIndexEnsurer struct {
	DBInterface
	errs      []error
	hangFirst bool
	calls     int
	colls     []string
	specs     []mongoapi.IndexSpec
	deadlines []time.Duration
}

func (m *mockIndexEnsurer) EnsureIndex(ctx context.Context, collName string, spec mongoapi.IndexSpec) error {
	m.calls++
	m.colls = append(m.colls, collName)
	m.specs = append(m.specs, spec)
	if deadline, ok := ctx.Deadline(); ok {
		m.deadlines = append(m.deadlines, time.Until(deadline))
	}
	if m.hangFirst && m.calls == 1 {
		<-ctx.Done()
		return ctx.Err()
	}
	if m.calls <= len(m.errs) {
		return m.errs[m.calls-1]
	}
	return nil
}

// concurrentCreationError is the error util's EnsureIndex returns when another
// process creates a conflicting index between its listing and its create.
func concurrentCreationError() error {
	return fmt.Errorf("create index %q on collection %s conflicts with an index created concurrently, retry: %w",
		testIndexSpec.Name, "amData", errors.New("IndexOptionsConflict"))
}

func TestEnsureIndexWithRetry(t *testing.T) {
	t.Run("succeeds first time with the spec unchanged", func(t *testing.T) {
		client := &mockIndexEnsurer{}
		err := ensureIndexWithRetry(client, "amData", testIndexSpec, time.Second, 5*time.Millisecond, time.Second)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if client.calls != 1 {
			t.Fatalf("expected 1 EnsureIndex call, got %d", client.calls)
		}
		if client.colls[0] != "amData" || client.specs[0].Name != testIndexSpec.Name {
			t.Fatalf("expected amData/%s, got %s/%s", testIndexSpec.Name, client.colls[0], client.specs[0].Name)
		}
	})

	t.Run("retries a transient error", func(t *testing.T) {
		client := &mockIndexEnsurer{errs: []error{errors.New("not primary")}}
		err := ensureIndexWithRetry(client, "amData", testIndexSpec, time.Second, 5*time.Millisecond, time.Second)
		if err != nil {
			t.Fatalf("expected retry to succeed, got %v", err)
		}
		if client.calls != 2 {
			t.Fatalf("expected 2 EnsureIndex calls, got %d", client.calls)
		}
	})

	t.Run("retries the concurrent-creation error", func(t *testing.T) {
		concurrent := concurrentCreationError()
		// The reason ensureIndexWithRetry does not reuse this predicate.
		if isRetryableIndexError(concurrent) {
			t.Fatal("isRetryableIndexError now accepts the concurrent-creation error; this test no longer shows why every error is retried")
		}
		client := &mockIndexEnsurer{errs: []error{concurrent}}
		err := ensureIndexWithRetry(client, "amData", testIndexSpec, time.Second, 5*time.Millisecond, time.Second)
		if err != nil {
			t.Fatalf("expected retry to succeed, got %v", err)
		}
		if client.calls != 2 {
			t.Fatalf("expected 2 EnsureIndex calls, got %d", client.calls)
		}
	})

	t.Run("returns the last error once the budget is spent", func(t *testing.T) {
		permanent := errors.New("not authorized")
		errs := make([]error, 1000)
		for i := range errs {
			errs[i] = permanent
		}
		client := &mockIndexEnsurer{errs: errs}
		err := ensureIndexWithRetry(client, "amData", testIndexSpec, 30*time.Millisecond, 5*time.Millisecond, time.Second)
		if err == nil {
			t.Fatal("expected an error once the budget is spent")
		}
		if !errors.Is(err, permanent) || !strings.Contains(err.Error(), "timed out ensuring index") {
			t.Fatalf("expected a timeout wrapping the last error, got %v", err)
		}
		if client.calls < 2 {
			t.Fatalf("expected the error to be retried within the budget, got %d calls", client.calls)
		}
	})

	t.Run("bounds a hung attempt by its own timeout", func(t *testing.T) {
		client := &mockIndexEnsurer{hangFirst: true}
		start := time.Now()
		err := ensureIndexWithRetry(client, "amData", testIndexSpec, 5*time.Second, 5*time.Millisecond, 20*time.Millisecond)
		if err != nil {
			t.Fatalf("expected the attempt after the hung one to succeed, got %v", err)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("hung attempt was not cut off by its timeout: took %s", elapsed)
		}
		if client.calls != 2 {
			t.Fatalf("expected 2 EnsureIndex calls, got %d", client.calls)
		}
		if client.deadlines[0] > 20*time.Millisecond {
			t.Fatalf("expected the attempt deadline to be at most 20ms, got %s", client.deadlines[0])
		}
	})
}

type clientWithoutEnsureIndex struct {
	DBInterface
}

func TestEnsureIndex(t *testing.T) {
	t.Run("nil client is an error", func(t *testing.T) {
		if err := EnsureIndex(nil, "amData", testIndexSpec); err == nil {
			t.Fatal("expected an error for a nil client")
		}
	})

	t.Run("client that cannot ensure indexes is an error", func(t *testing.T) {
		err := EnsureIndex(&clientWithoutEnsureIndex{}, "amData", testIndexSpec)
		if err == nil || !strings.Contains(err.Error(), "cannot ensure index") {
			t.Fatalf("expected a cannot-ensure error, got %v", err)
		}
	})

	t.Run("delegates to the client", func(t *testing.T) {
		client := &mockIndexEnsurer{}
		if err := EnsureIndex(client, "amData", testIndexSpec); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if client.calls != 1 {
			t.Fatalf("expected 1 EnsureIndex call, got %d", client.calls)
		}
		if client.deadlines[0] > indexEnsureAttemptTimeout {
			t.Fatalf("expected an attempt deadline of at most %s, got %s", indexEnsureAttemptTimeout, client.deadlines[0])
		}
	})
}
