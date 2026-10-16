// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func authenticationSubscription() *models.AuthenticationSubscription {
	sqn := models.NewSequenceNumber()
	sqn.SetSqn(testSequenceNumber)
	authSubs := models.NewAuthenticationSubscription(models.AUTHMETHOD__5_G_AKA)
	authSubs.SetAuthenticationManagementField("8000")
	authSubs.SetEncOpcKey("8e27b6af0e692e750f32667a3b14605d")       // Required
	authSubs.SetEncPermanentKey("8baf473f2f8fd09487cccbd7097c6862") // Required
	authSubs.SetSequenceNumber(*sqn)
	return authSubs
}

const testAuthDbName = "auth_test_db"

func setupTestFactory() func() {
	origConfig := factory.WebUIConfig
	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			Mongodb: &factory.Mongodb{
				AuthKeysDbName: testAuthDbName,
			},
		},
	}
	return func() {
		factory.WebUIConfig = origConfig
	}
}

type txMockDB struct {
	dbadapter.DBInterface
	receivedPost    []map[string]any
	receivedPut     []map[string]any
	receivedDelete  []map[string]any
	receivedGet     []map[string]any
	receivedReplace []map[string]any
	receivedUpdate  []map[string]any
	postErr         error
	putErr          error
	deleteErr       error
	getErr          error
	getResult       map[string]any
	replaceErr      error
	replaceMatched  bool
	updateErr       error
	updateMatched   bool
}

func (m *txMockDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	m.receivedGet = append(m.receivedGet, map[string]any{
		collKey:   collName,
		filterKey: filter,
	})
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResult, nil
}

func (m *txMockDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.postErr != nil {
		return false, m.postErr
	}
	m.receivedPost = append(m.receivedPost, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   postData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.putErr != nil {
		return false, m.putErr
	}
	m.receivedPut = append(m.receivedPut, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   putData,
	})
	return true, nil
}

func (m *txMockDB) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.receivedDelete = append(m.receivedDelete, map[string]any{
		collKey:   collName,
		filterKey: filter,
	})
	return nil
}

func (m *txMockDB) RestfulAPIReplaceOneNoUpsert(collName string, filter bson.M, replacement map[string]any) (bool, error) {
	if m.replaceErr != nil {
		return false, m.replaceErr
	}
	m.receivedReplace = append(m.receivedReplace, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   replacement,
	})
	return m.replaceMatched, nil
}

func (m *txMockDB) RestfulAPIUpdateOneNoUpsert(collName string, filter bson.M, updateData map[string]any) (bool, error) {
	if m.updateErr != nil {
		return false, m.updateErr
	}
	m.receivedUpdate = append(m.receivedUpdate, map[string]any{
		collKey:   collName,
		filterKey: filter,
		dataKey:   updateData,
	})
	return m.updateMatched, nil
}

// authSubsDataColl writes/deletes must go through AuthDBClient (its own server) while amData
// goes through CommonDBClient, as two independent non-transactional calls (see
// subscriberAuthenticationDataCreate/Update/Delete).
func TestSubscriberAuthenticationDataCreate_Success(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(authMock.receivedPost) != 1 {
		t.Fatalf("expected 1 auth DB post call, got %d", len(authMock.receivedPost))
	}
	if authMock.receivedPost[0][collKey] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, authMock.receivedPost[0][collKey])
	}
	if len(commonMock.receivedPost) != 1 {
		t.Fatalf("expected 1 amData post call, got %d", len(commonMock.receivedPost))
	}
	if commonMock.receivedPost[0][collKey] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, commonMock.receivedPost[0][collKey])
	}

	expectedFilter := bson.M{ueIdKey: testSubscriberImsi}
	if !reflect.DeepEqual(authMock.receivedPost[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authMock.receivedPost[0][filterKey])
	}
	if !reflect.DeepEqual(commonMock.receivedPost[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonMock.receivedPost[0][filterKey])
	}
}

func TestSubscriberAuthenticationDataCreate_AuthDBFails(t *testing.T) {
	authMock := &txMockDB{postErr: fmt.Errorf("auth db failure")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(commonMock.receivedPost) != 0 {
		t.Error("expected no CommonDB write when auth write fails")
	}
}

func TestSubscriberAuthenticationDataCreate_CommonDBFails(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{postErr: errors.New(testCommonDBErrMsg)}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(authMock.receivedPost) != 1 {
		t.Error("expected auth write to have been attempted")
	}
	if len(authMock.receivedDelete) != 1 {
		t.Fatalf("expected auth write to be rolled back, got %d delete calls", len(authMock.receivedDelete))
	}
	postedAuthData := authMock.receivedPost[0][dataKey].(map[string]any)
	if !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], bson.M(postedAuthData)) {
		t.Errorf("expected rollback filter %v, got %v", bson.M(postedAuthData), authMock.receivedDelete[0][filterKey])
	}
}

// If amData actually exists on re-check -- either because the "failed" write was applied despite
// the reported error, or because a concurrent create already wrote it -- the credential must not
// be rolled back: it belongs to whichever create produced the amData record that is now present.
func TestSubscriberAuthenticationDataCreate_CommonDBFails_AmDataAlreadyExists(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{
		postErr:   errors.New(testCommonDBErrMsg),
		getResult: map[string]any{ueIdKey: testSubscriberImsi},
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no rollback when amData already exists, got %d delete calls", len(authMock.receivedDelete))
	}
}

// If the state of amData can't be verified after a create failure, the credential must be left
// alone rather than risk deleting a concurrent request's still-valid write.
func TestSubscriberAuthenticationDataCreate_CommonDBFails_VerifyReadFails(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{
		postErr: errors.New(testCommonDBErrMsg),
		getErr:  fmt.Errorf("cannot reach common db"),
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil || err.Error() != testCommonDBErrMsg {
		t.Errorf("expected original amData error to be returned, got %v", err)
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no rollback when amData state cannot be verified, got %d delete calls", len(authMock.receivedDelete))
	}
}

// A timeout proves nothing about whether amData actually committed -- it may still land after
// this call returns -- so even a nil result from the existence check would not be trustworthy.
// Compensation must be skipped entirely, never attempted based on a racy read.
func TestSubscriberAuthenticationDataCreate_CommonDBTimeout_RetainsCredential(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{
		postErr: fmt.Errorf("RestfulAPIPutOneWithContext UpdateOne err: %w", context.DeadlineExceeded),
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if len(commonMock.receivedGet) != 0 {
		t.Errorf("expected no existence check on an ambiguous timeout, got %d get calls", len(commonMock.receivedGet))
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no rollback on an ambiguous timeout, got %d delete calls", len(authMock.receivedDelete))
	}
	if len(authMock.receivedReplace) != 0 {
		t.Errorf("expected no restore on an ambiguous timeout, got %d replace calls", len(authMock.receivedReplace))
	}
}

// A network error (e.g. a connection reset while awaiting the response) is just as ambiguous as
// a timeout: it proves nothing about whether the write committed, so compensation must be skipped
// here too.
func TestSubscriberAuthenticationDataCreate_CommonDBNetworkError_RetainsCredential(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{
		postErr: fmt.Errorf("RestfulAPIPutOneWithContext UpdateOne err: %w", mongo.CommandError{Labels: []string{"NetworkError"}}),
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if len(commonMock.receivedGet) != 0 {
		t.Errorf("expected no existence check on an ambiguous network error, got %d get calls", len(commonMock.receivedGet))
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no rollback on an ambiguous network error, got %d delete calls", len(authMock.receivedDelete))
	}
	if len(authMock.receivedReplace) != 0 {
		t.Errorf("expected no restore on an ambiguous network error, got %d replace calls", len(authMock.receivedReplace))
	}
}

func TestSubscriberAuthenticationDataCreate_CommonDBFails_RollbackAlsoFails(t *testing.T) {
	authMock := &txMockDB{deleteErr: fmt.Errorf("rollback failure")}
	commonMock := &txMockDB{postErr: errors.New(testCommonDBErrMsg)}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if err.Error() != testCommonDBErrMsg {
		t.Errorf("expected original amData error to be returned, got %v", err)
	}
}

// If the pre-create read of the credential cannot be verified, the create must not proceed:
// without knowing whether a prior document existed, a later failure could not safely choose
// between restoring it and deleting it outright.
func TestSubscriberAuthenticationDataCreate_PreviousAuthReadFails(t *testing.T) {
	authMock := &txMockDB{getErr: fmt.Errorf("cannot reach auth db")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(authMock.receivedPost) != 0 {
		t.Error("expected no auth write when the pre-create read fails")
	}
}

// A credential retained from before this call (e.g. left behind by device-group/slice cleanup
// removing amData but not the credential) must be restored, not deleted outright, if the amData
// write fails: this call's upsert overwrote that document in place, so deleting it on rollback
// would destroy data that predates this call entirely.
func TestSubscriberAuthenticationDataCreate_CommonDBFails_RestoresPreviousCredential(t *testing.T) {
	previousAuth := map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	authMock := &txMockDB{getResult: previousAuth, updateMatched: true, replaceMatched: true}
	commonMock := &txMockDB{postErr: errors.New(testCommonDBErrMsg)}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no delete when a previous credential must be restored, got %d delete calls", len(authMock.receivedDelete))
	}
	if len(authMock.receivedReplace) != 1 {
		t.Fatalf("expected 1 restore (replace) call, got %d", len(authMock.receivedReplace))
	}
	if len(authMock.receivedUpdate) != 1 {
		t.Fatalf("expected 1 auth overwrite (update) call, got %d", len(authMock.receivedUpdate))
	}
	writtenAuthData := authMock.receivedUpdate[0][dataKey].(map[string]any)
	if !reflect.DeepEqual(authMock.receivedReplace[0][filterKey], bson.M(writtenAuthData)) {
		t.Errorf("expected restore filter %v, got %v", bson.M(writtenAuthData), authMock.receivedReplace[0][filterKey])
	}
	if !reflect.DeepEqual(authMock.receivedReplace[0][dataKey], previousAuth) {
		t.Errorf("expected restore data %v, got %v", previousAuth, authMock.receivedReplace[0][dataKey])
	}
}

// A retained credential can carry fields this create's own write never touches (e.g. extra data
// left by another writer before this call). The restore's compare-and-swap filter must include
// them too, not just the fields authDataBsonA happens to carry: otherwise a concurrent change to
// one of previousAuth's OTHER fields would go undetected and the stale snapshot would still get
// restored, discarding that change.
func TestSubscriberAuthenticationDataCreate_CommonDBFails_RestoreFilterIncludesUnrelatedFields(t *testing.T) {
	previousAuth := map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne, "extraField": "preexisting"}
	authMock := &txMockDB{getResult: previousAuth, updateMatched: true, replaceMatched: true}
	commonMock := &txMockDB{postErr: errors.New(testCommonDBErrMsg)}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected the original amData error but got nil")
	}
	if len(authMock.receivedReplace) != 1 {
		t.Fatalf("expected 1 restore (replace) call, got %d", len(authMock.receivedReplace))
	}
	filter, ok := authMock.receivedReplace[0][filterKey].(bson.M)
	if !ok {
		t.Fatalf("expected filter of type bson.M, got %T", authMock.receivedReplace[0][filterKey])
	}
	if filter["extraField"] != "preexisting" {
		t.Errorf("expected restore filter to include previousAuth's unrelated field, got %v", filter)
	}
}

// If the filter no longer matches by the time the restore runs -- because the document changed
// since this call's own write, whether from a concurrent webconsole call or an out-of-band update
// such as a UDR sequence number re-synchronisation -- RestfulAPIReplaceOneNoUpsert reports zero
// matches rather than inserting a stale document: the restore must treat that as a harmless
// no-op, not an error.
func TestSubscriberAuthenticationDataCreate_CommonDBFails_RestoreGenerationMismatchIsNoOp(t *testing.T) {
	previousAuth := map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	authMock := &txMockDB{getResult: previousAuth, updateMatched: true, replaceMatched: false}
	commonMock := &txMockDB{postErr: errors.New(testCommonDBErrMsg)}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil || err.Error() != testCommonDBErrMsg {
		t.Errorf("expected original amData error to be returned, got %v", err)
	}
	if len(authMock.receivedReplace) != 1 {
		t.Fatalf("expected the restore to still be attempted once, got %d calls", len(authMock.receivedReplace))
	}
	if len(authMock.receivedDelete) != 0 {
		t.Errorf("expected no delete fallback on a generation mismatch, got %d delete calls", len(authMock.receivedDelete))
	}
}

// Reproduces an older, delayed auth write landing after a newer operation already changed the
// document: the conditional overwrite must detect the mismatch and abort before ever touching
// amData, instead of silently clobbering whatever is there now.
func TestSubscriberAuthenticationDataCreate_ConcurrentModificationAborts(t *testing.T) {
	previousAuth := map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	authMock := &txMockDB{getResult: previousAuth, updateMatched: false}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataCreate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if len(commonMock.receivedPost) != 0 {
		t.Errorf("expected no amData write once the auth overwrite is aborted, got %d post calls", len(commonMock.receivedPost))
	}
}

func TestSubscriberAuthenticationDataUpdate_Success(t *testing.T) {
	currentAuth := map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	authMock := &txMockDB{getResult: currentAuth, updateMatched: true}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(authMock.receivedUpdate) != 1 {
		t.Fatalf("expected 1 auth DB update call, got %d", len(authMock.receivedUpdate))
	}
	if authMock.receivedUpdate[0][collKey] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, authMock.receivedUpdate[0][collKey])
	}
	// Conditioned on the exact document just read, so a concurrent webconsole write or an
	// out-of-band update (e.g. a UDR sequence number re-synchronisation) is detected as a mismatch
	// instead of silently overwritten.
	if !reflect.DeepEqual(authMock.receivedUpdate[0][filterKey], bson.M(currentAuth)) {
		t.Errorf("expected filter %v, got %v", bson.M(currentAuth), authMock.receivedUpdate[0][filterKey])
	}
	// The amData record was already confirmed to exist by the caller, so update must not
	// re-write it: there's nothing to change and it would only add a failure point.
	if len(commonMock.receivedPut) != 0 {
		t.Error("expected no CommonDB write on update")
	}
}

// If the credential doesn't exist yet, update repairs it with a direct insert instead of
// aborting: the caller has already confirmed the subscriber (amData) exists, so this is not a
// "no such subscriber" case -- it's a credential missing from AuthDBClient entirely (e.g. a
// subscriber provisioned before issue #585 was fixed, whose amData was written but whose
// credential landed on the wrong server).
func TestSubscriberAuthenticationDataUpdate_RepairsMissingCredential(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(authMock.receivedPost) != 1 {
		t.Fatalf("expected 1 auth DB post (repair) call, got %d", len(authMock.receivedPost))
	}
	if len(authMock.receivedUpdate) != 0 {
		t.Errorf("expected no conditional update when there is nothing to condition on, got %d calls", len(authMock.receivedUpdate))
	}
}

func TestSubscriberAuthenticationDataUpdate_RepairFails(t *testing.T) {
	authMock := &txMockDB{postErr: fmt.Errorf("auth db failure")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestSubscriberAuthenticationDataUpdate_ReadFails(t *testing.T) {
	authMock := &txMockDB{getErr: fmt.Errorf("auth db failure")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestSubscriberAuthenticationDataUpdate_AuthDBFails(t *testing.T) {
	authMock := &txMockDB{
		getResult: map[string]any{ueIdKey: testSubscriberImsi},
		updateErr: fmt.Errorf("auth db failure"),
	}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// A stale or concurrent write that no longer matches the document just read must abort cleanly
// instead of silently upserting (the write already has upsert disabled; this proves the caller
// also treats a reported non-match as an error).
func TestSubscriberAuthenticationDataUpdate_ConcurrentModificationAborts(t *testing.T) {
	authMock := &txMockDB{
		getResult:     map[string]any{ueIdKey: testSubscriberImsi},
		updateMatched: false,
	}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	subsData := authenticationSubscription()
	err := subscriberAuthenticationDataUpdate(testSubscriberImsi, subsData)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestSubscriberAuthenticationDataDelete_Success(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(authMock.receivedDelete) != 1 {
		t.Fatalf("expected 1 auth DB delete call, got %d", len(authMock.receivedDelete))
	}
	if authMock.receivedDelete[0][collKey] != authSubsDataColl {
		t.Errorf("expected collection %q, got %q", authSubsDataColl, authMock.receivedDelete[0][collKey])
	}
	if len(commonMock.receivedDelete) != 1 {
		t.Fatalf("expected 1 amData delete call, got %d", len(commonMock.receivedDelete))
	}
	if commonMock.receivedDelete[0][collKey] != amDataColl {
		t.Errorf("expected collection %q, got %q", amDataColl, commonMock.receivedDelete[0][collKey])
	}

	expectedFilter := bson.M{ueIdKey: testSubscriberImsi, provisioningTokenKey: bson.M{"$exists": false}}
	if !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, authMock.receivedDelete[0][filterKey])
	}
	if !reflect.DeepEqual(commonMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected filter %v, got %v", expectedFilter, commonMock.receivedDelete[0][filterKey])
	}
}

// amData is deleted first; if that later auth delete fails, amData is already gone so the
// partial state is self-healing (recreation is allowed and overwrites the stale credential).
func TestSubscriberAuthenticationDataDelete_AuthDBDeleteFails(t *testing.T) {
	authMock := &txMockDB{deleteErr: fmt.Errorf("fail on authdb delete")}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(commonMock.receivedDelete) != 1 {
		t.Error("expected amData delete to have been attempted before the failing auth delete")
	}
}

// amData delete failing first must leave the credential untouched.
func TestSubscriberAuthenticationDataDelete_CommonDBDeleteFails(t *testing.T) {
	authMock := &txMockDB{}
	commonMock := &txMockDB{deleteErr: fmt.Errorf("fail on commondb delete")}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	err := subscriberAuthenticationDataDelete(testSubscriberImsi)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if len(authMock.receivedDelete) != 0 {
		t.Error("expected no auth delete when amData delete fails")
	}
}

// Both deletes must be scoped to each collection's own current generation, so that if a
// concurrent recreate replaces the subscriber (writing a new provisioningToken) while this delete
// is between its two steps, neither the recreated credential nor its amData record is removed.
func TestSubscriberAuthenticationDataDelete_ScopesToCapturedGeneration(t *testing.T) {
	authMock := &txMockDB{
		getResult: map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne},
	}
	commonMock := &txMockDB{
		getResult: map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne},
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	if err := subscriberAuthenticationDataDelete(testSubscriberImsi); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedFilter := bson.M{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	if len(commonMock.receivedDelete) != 1 || !reflect.DeepEqual(commonMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected amData delete filter %v, got %v", expectedFilter, commonMock.receivedDelete[0][filterKey])
	}
	if len(authMock.receivedDelete) != 1 || !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected auth delete filter %v, got %v", expectedFilter, authMock.receivedDelete[0][filterKey])
	}
}

// A retry after the auth delete fails must still find and remove the credential by re-reading its
// current token, even though amData was already removed by the first (partially successful)
// attempt.
func TestSubscriberAuthenticationDataDelete_RetryAfterAuthDeleteFailureStillSucceeds(t *testing.T) {
	authMock := &txMockDB{
		getResult: map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne},
		deleteErr: fmt.Errorf("transient auth db failure"),
	}
	commonMock := &txMockDB{}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	if err := subscriberAuthenticationDataDelete(testSubscriberImsi); err == nil {
		t.Fatal("expected the first delete attempt to fail")
	}
	if len(authMock.receivedDelete) != 0 {
		t.Fatalf("expected the failed attempt to record no delete, got %d", len(authMock.receivedDelete))
	}

	authMock.deleteErr = nil
	if err := subscriberAuthenticationDataDelete(testSubscriberImsi); err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if len(authMock.receivedDelete) != 1 {
		t.Fatalf("expected 1 auth delete call from the retry, got %d", len(authMock.receivedDelete))
	}
	expectedFilter := bson.M{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	if !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], expectedFilter) {
		t.Errorf("expected retry delete filter %v, got %v", expectedFilter, authMock.receivedDelete[0][filterKey])
	}
}

// An orphaned amData record (e.g. left behind by a prior attempt whose amData write succeeded
// despite a client-observed error, after which the credential was rolled back) must be cleaned up
// using its own generation, not the credential's: the credential is already gone here, so deriving
// amData's filter from its snapshot (no token) would produce an exists:false filter that cannot
// match amData's own, still-tokened document.
func TestSubscriberAuthenticationDataDelete_OrphanedAmDataUsesItsOwnGeneration(t *testing.T) {
	authMock := &txMockDB{getResult: nil}
	commonMock := &txMockDB{
		getResult: map[string]any{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne},
	}
	origAuthDB := dbadapter.AuthDBClient
	origCommonDB := dbadapter.CommonDBClient
	defer func() {
		dbadapter.AuthDBClient = origAuthDB
		dbadapter.CommonDBClient = origCommonDB
	}()
	dbadapter.AuthDBClient = authMock
	dbadapter.CommonDBClient = commonMock

	if err := subscriberAuthenticationDataDelete(testSubscriberImsi); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedAmDataFilter := bson.M{ueIdKey: testSubscriberImsi, provisioningTokenKey: testGenerationOne}
	if len(commonMock.receivedDelete) != 1 || !reflect.DeepEqual(commonMock.receivedDelete[0][filterKey], expectedAmDataFilter) {
		t.Errorf("expected amData delete filter %v, got %v", expectedAmDataFilter, commonMock.receivedDelete[0][filterKey])
	}
	expectedAuthFilter := bson.M{ueIdKey: testSubscriberImsi, provisioningTokenKey: bson.M{"$exists": false}}
	if len(authMock.receivedDelete) != 1 || !reflect.DeepEqual(authMock.receivedDelete[0][filterKey], expectedAuthFilter) {
		t.Errorf("expected auth delete filter %v, got %v", expectedAuthFilter, authMock.receivedDelete[0][filterKey])
	}
}

// Locking one IMSI must not block a different IMSI mapping to a different shard, otherwise
// provisioning would still serialize unrelated subscribers as if there were one global lock.
func TestLockSubscriber_DifferentIMSIsDoNotShareAShard(t *testing.T) {
	const otherImsi = "imsi-208930100007488" // hashes to a different shard than testSubscriberImsi
	unlockFirst := lockSubscriber(testSubscriberImsi)
	defer unlockFirst()

	done := make(chan struct{})
	go func() {
		lockSubscriber(otherImsi)()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected a different IMSI's lock to be acquired immediately, but it blocked")
	}
}

func Test_handleSubscriberGet(t *testing.T) {
	origAuthDBClient := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuthDBClient }()
	subscriber := authenticationSubscription()
	dbadapter.AuthDBClient = &AuthDBMockDBClient{subscribers: []string{testSubscriberImsi}}
	subscriberResult := subscriberAuthenticationDataGet(testSubscriberImsi)
	if !reflect.DeepEqual(subscriber, subscriberResult) {
		t.Errorf("expected subscriber %v, got %v", subscriber, subscriberResult)
	}
}

// This write path fetches a device group straight from storage rather than through
// deviceGroupPostHelper, so it must relabel a rate unit left over from before that helper stored
// it as bps -- otherwise it persists the stale unit rather than correcting it.
func TestUpdateSubscriberInDeviceGroups_RelabelsStaleRateUnit(t *testing.T) {
	origDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origDBClient }()
	group := deviceGroup(testGroupName)
	mock := &DeviceGroupMockDBClient{configuredDeviceGroups: []configmodels.DeviceGroups{group}}
	dbadapter.CommonDBClient = mock

	statusCode, err := updateSubscriberInDeviceGroups(group.Imsis[0])
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, statusCode)
	}
	if len(mock.postData) == 0 {
		t.Fatal("expected the device group to be stored")
	}
	var stored configmodels.DeviceGroups
	if err := json.Unmarshal(configmodels.MapToByte(mock.postData[0][dataKey].(map[string]any)), &stored); err != nil {
		t.Fatalf("failed to unmarshal the stored device group: %v", err)
	}
	qos := stored.IpDomainsExpanded[0].UeDnnQos
	if qos.BitrateUnit != bitrateUnitBps {
		t.Errorf("stored bitrate-unit = %q, want %q", qos.BitrateUnit, bitrateUnitBps)
	}
	// The rates themselves are untouched: this path does not re-run convertToBps.
	original := group.IpDomainsExpanded[0].UeDnnQos
	if qos.DnnMbrUplink != original.DnnMbrUplink || qos.DnnMbrDownlink != original.DnnMbrDownlink {
		t.Errorf("rates changed: got %d/%d, want %d/%d", qos.DnnMbrUplink, qos.DnnMbrDownlink, original.DnnMbrUplink, original.DnnMbrDownlink)
	}
}
