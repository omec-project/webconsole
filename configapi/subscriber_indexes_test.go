// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"context"
	"slices"
	"testing"

	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/dbadapter"
)

type expectedIndex struct {
	collName string
	name     string
	keys     []string
}

// These are transcribed from the UDR, which is what reads these collections and
// which webconsole cannot import: producer/data_repository.go on
// omec-project/udr main. The collNames of the smData and policy handlers are
// literals there, so they carry the handler's name instead. They are
// deliberately not webconsole's own constants: an index on a spelling the UDR
// does not use seeks nothing.
const (
	udrCollAmData                       = "subscriptionData.provisionedData.amData"
	udrCollSmfSelectionSubscriptionData = "subscriptionData.provisionedData.smfSelectionSubscriptionData"
	udrCollAuthenticationSubscription   = "subscriptionData.authenticationData.authenticationSubscription"
	udrHandleQuerySmDataCollName        = "subscriptionData.provisionedData.smData"
	udrPolicyAmDataGetCollName          = "policyData.ues.amData"
	udrPolicySmDataGetCollName          = "policyData.ues.smData"
	udrParamUeId                        = "ueId"
	udrParamServingPlmnId               = "servingPlmnId"
)

var (
	expectedCommonDBIndexes = []expectedIndex{
		{udrCollAmData, "amDataByUeIdAndServingPlmnId", []string{udrParamUeId, udrParamServingPlmnId}},
		{udrHandleQuerySmDataCollName, "smDataByUeIdAndServingPlmnId", []string{udrParamUeId, udrParamServingPlmnId}},
		{udrCollSmfSelectionSubscriptionData, "smfSelectionSubscriptionDataByUeIdAndServingPlmnId", []string{udrParamUeId, udrParamServingPlmnId}},
		{udrPolicyAmDataGetCollName, "amPolicyDataByUeId", []string{udrParamUeId}},
		{udrPolicySmDataGetCollName, "smPolicyDataByUeId", []string{udrParamUeId}},
	}
	expectedAuthDBIndexes = []expectedIndex{
		{udrCollAuthenticationSubscription, "authenticationSubscriptionByUeId", []string{udrParamUeId}},
	}
)

func checkIndexes(t *testing.T, db string, got []collectionIndex, want []expectedIndex) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: expected %d indexes, got %d", db, len(want), len(got))
	}
	for i, w := range want {
		g := got[i]
		if g.collName != w.collName {
			t.Errorf("%s index %d: expected collection %q, got %q", db, i, w.collName, g.collName)
		}
		if g.spec.Name != w.name {
			t.Errorf("%s index on %s: expected name %q, got %q", db, w.collName, w.name, g.spec.Name)
		}
		keys := make([]string, 0, len(g.spec.Keys))
		for _, k := range g.spec.Keys {
			keys = append(keys, k.Key)
			if k.Value != 1 {
				t.Errorf("%s index %q: key %q has direction %v, want 1", db, g.spec.Name, k.Key, k.Value)
			}
		}
		if !slices.Equal(keys, w.keys) {
			t.Errorf("%s index %q: expected keys %v, got %v", db, g.spec.Name, w.keys, keys)
		}
		if g.spec.Unique || g.spec.Sparse || g.spec.PartialFilter != nil {
			t.Errorf("%s index %q: expected a plain non-unique index, got %+v", db, g.spec.Name, g.spec)
		}
	}
}

func TestSubscriberIndexSpecs(t *testing.T) {
	checkIndexes(t, "common DB", commonDBSubscriberIndexes, expectedCommonDBIndexes)
	checkIndexes(t, "auth DB", authDBSubscriberIndexes, expectedAuthDBIndexes)
}

type recordingIndexClient struct {
	dbadapter.DBInterface
	ensured []collectionIndex
}

func (m *recordingIndexClient) EnsureIndex(ctx context.Context, collName string, spec mongoapi.IndexSpec) error {
	m.ensured = append(m.ensured, collectionIndex{collName, spec})
	return nil
}

func swapDBClients(t *testing.T, common, auth dbadapter.DBInterface) {
	t.Helper()
	originalCommon, originalAuth := dbadapter.CommonDBClient, dbadapter.AuthDBClient
	dbadapter.CommonDBClient, dbadapter.AuthDBClient = common, auth
	t.Cleanup(func() {
		dbadapter.CommonDBClient, dbadapter.AuthDBClient = originalCommon, originalAuth
	})
}

func TestEnsureSubscriberIndexes(t *testing.T) {
	t.Run("ensures each index once, on the client its readers use", func(t *testing.T) {
		common, auth := &recordingIndexClient{}, &recordingIndexClient{}
		swapDBClients(t, common, auth)

		if err := EnsureSubscriberIndexes(); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		checkIndexes(t, "common DB", common.ensured, expectedCommonDBIndexes)
		checkIndexes(t, "auth DB", auth.ensured, expectedAuthDBIndexes)
	})

	t.Run("a client that cannot ensure indexes stops startup", func(t *testing.T) {
		auth := &recordingIndexClient{}
		swapDBClients(t, &MockMongoClientEmptyDB{}, auth)

		if err := EnsureSubscriberIndexes(); err == nil {
			t.Fatal("expected an error for a common client without EnsureIndex")
		}
		if len(auth.ensured) != 0 {
			t.Fatalf("expected no index ensured after the failure, got %d", len(auth.ensured))
		}
	})

	t.Run("a missing auth client stops startup", func(t *testing.T) {
		swapDBClients(t, &recordingIndexClient{}, nil)

		if err := EnsureSubscriberIndexes(); err == nil {
			t.Fatal("expected an error for a nil auth client")
		}
	})
}
