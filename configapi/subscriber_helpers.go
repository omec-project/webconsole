// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// subscriberAuthLockShards bounds per-IMSI locking to a fixed set of mutexes (striping), so
// provisioning one subscriber no longer blocks unrelated subscribers while avoiding an
// ever-growing map of per-IMSI locks.
const subscriberAuthLockShards = 64

var subscriberAuthLocks [subscriberAuthLockShards]sync.Mutex

// lockSubscriber serializes PostSubscriberByID/PutSubscriberByID/DeleteSubscriberByID for a given
// IMSI: the existence-check-then-write sequences below are not atomic across AuthDBClient and
// CommonDBClient, so two requests for the same IMSI must never interleave. Call the returned
// function to release the lock.
//
// This is an in-process lock only: it does not fence concurrent writes from a second webconsole
// instance sharing the same databases. webconsole has no multi-instance/HA deployment support
// today (no distributed lock, leader election, or database-enforced per-IMSI claim), so this
// matches the existing assumption already made by device_group_helpers.go's package-level rwLock;
// revisit both if that changes.
func lockSubscriber(imsi string) (unlock func()) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(imsi))
	shard := &subscriberAuthLocks[h.Sum32()%subscriberAuthLockShards]
	shard.Lock()
	return shard.Unlock
}

func subscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	filter := bson.M{ueIdKey: imsi}
	authSubDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		return
	}
	err = json.Unmarshal(configmodels.MapToByte(authSubDataInterface), &authSubData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall subscriber %+v", authSubDataInterface)
		return
	}
	return authSubData
}

// authSubsDataColl lives on AuthDBClient's server (mongodb.authUrl), which can differ from
// CommonDBClient's (mongodb.url). A mongo transaction can't span two different clients/servers,
// so each collection is written through its own client as a separate, non-transactional call.
// Ordering and compensation below are chosen so that any partial failure leaves the system in a
// state that a retry can safely self-heal (writes are upserts keyed on imsi).
func subscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	// device-group/slice sync (device_group_helpers.go, slice_helpers.go) already hold rwLock.Lock()
	// across their own credential read and amData write; taking the read lock here excludes this
	// call's compensation below from running concurrently with that, so sync can never observe (or
	// write amData for) a credential this call is in the middle of rolling back or restoring.
	rwLock.RLock()
	defer rwLock.RUnlock()

	filter := bson.M{ueIdKey: imsi}
	logger.WebUILog.Infof("%+v", authSubData)
	// Captured before the upsert below so a failed amData write can restore this exact prior
	// document instead of deleting it outright: the upsert below can land on a credential this
	// call did not create (e.g. one retained by device-group/slice cleanup removing amData but
	// not the credential), and that pre-existing document must survive a failed create.
	previousAuth, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to read authentication subscription before create for %s: %+v", imsi, err)
		return err
	}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA[ueIdKey] = imsi
	// provisioningTokenKey ties this call's auth and amData writes to one generation, so if the
	// amData write below fails, compensation can be scoped to only the credential this call just
	// wrote and never a concurrent create's newer one.
	token := uuid.NewString()
	authDataBsonA[provisioningTokenKey] = token
	basicAmData := map[string]any{ueIdKey: imsi, provisioningTokenKey: token}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if previousAuth == nil {
		// Nothing existed before this call: a plain insert is safe, there is nothing to conflict with.
		// This is still an upsert, not a DB-enforced insert-only write: authSubsDataColl intentionally
		// carries no unique index on ueId (see commonDBSubscriberIndexes/authDBSubscriberIndexes in
		// subscriber_indexes.go) because a unique index over a collection already holding a legacy
		// duplicate could never be created, which would stop webconsole from starting. Two concurrent
		// creates for the same brand-new IMSI are therefore only serialized by lockSubscriber, which is
		// in-process only (see its doc comment).
		if _, err := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA); err != nil {
			logger.DbLog.Errorf("failed to create authentication subscription error: %+v", err)
			return err
		}
	} else {
		// A document already existed (e.g. a credential retained by device-group/slice cleanup, or an
		// amData-only orphan left by a prior ambiguous write -- see PostSubscriberByID). Overwrite it
		// only if it still matches exactly what was just read: a blind upsert here could otherwise
		// silently clobber a concurrent webconsole write, or an out-of-band update such as a UDR
		// sequence number re-synchronisation, that landed between that read and this write.
		matched, err := dbadapter.AuthDBClient.RestfulAPIUpdateOneNoUpsert(authSubsDataColl, bson.M(previousAuth), authDataBsonA)
		if err != nil {
			logger.DbLog.Errorf("failed to create authentication subscription error: %+v", err)
			return err
		}
		if !matched {
			err := fmt.Errorf("authentication subscription for %s changed concurrently; aborting create", imsi)
			logger.DbLog.Errorf("%v", err)
			return err
		}
	}
	logger.WebUILog.Infof("created authentication subscription in authenticationSubscription collection: %s", imsi)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson); err != nil {
		logger.DbLog.Errorf("failed to create amData error: %+v", err)
		if mongo.IsTimeout(err) || mongo.IsNetworkError(err) {
			// Neither a timeout nor a network error (e.g. a connection reset while awaiting the
			// response) proves whether the server actually committed the write: it may still land
			// after this call returns. Reading amData now and seeing it absent would not be proof
			// either (the write could commit moments later), so compensating here could delete or
			// overwrite a credential that is about to get its matching amData -- leaving amData
			// without its intended authentication data instead. Retain the credential as-is and let
			// the caller's error path surface this as needing manual reconciliation or a retry.
			logger.DbLog.Warnf("amData create for %s has an ambiguous outcome; retaining authentication subscription as-is: %+v", imsi, err)
			return err
		}
		// The write may have actually been applied despite the reported error (e.g. a concurrent
		// create already having written amData for this IMSI). Once amData exists there is nothing to
		// roll back.
		existingAmData, checkErr := dbadapter.CommonDBClient.RestfulAPIGetOne(amDataColl, filter)
		if checkErr != nil {
			logger.DbLog.Errorf("failed to verify amData state for %s after create failure; manual reconciliation required: %+v", imsi, checkErr)
			return err
		}
		if existingAmData != nil {
			logger.DbLog.Warnf("amData for %s already exists despite create error; skipping authentication subscription rollback", imsi)
			return err
		}
		// amData is confirmed absent: compensate, scoped to the exact content this call wrote (not just
		// imsi/provisioningToken) so any out-of-band change since then -- a concurrent webconsole write,
		// or an external update such as a UDR sequence number re-synchronisation -- is detected and
		// treated as a no-op instead of being discarded.
		writtenFilter := authDataBsonA
		if previousAuth == nil {
			// No credential existed before this call: the one we just wrote is entirely ours, and the live
			// document is exactly authDataBsonA (a fresh insert, not a merge), so filtering on it alone
			// already detects a change to any field. Safe to delete outright.
			if rollbackErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, writtenFilter); rollbackErr != nil {
				logger.DbLog.Errorf("failed to roll back authentication subscription for %s after amData create failure; manual reconciliation required: %+v", imsi, rollbackErr)
			}
		} else {
			// A credential already existed and RestfulAPIUpdateOneNoUpsert's $set merged authDataBsonA onto
			// it: the live document is previousAuth's fields overlaid with authDataBsonA's, not
			// authDataBsonA alone. Filtering on just authDataBsonA's subset would still match even if a
			// concurrent writer changed one of previousAuth's OTHER fields in the meantime, and the
			// ReplaceOne below would then silently discard that change along with this call's own.
			// Filtering on the full reconstructed document instead catches a change to ANY field.
			writtenFilter = mergeBsonFields(previousAuth, authDataBsonA)
			// Replace it back to its prior content (not a $set-merge, and never an upsert) so a legitimate
			// pre-existing credential never disappears -- nor keeps stray fields this call's write added --
			// because of this call's failure. If the filter no longer matches (something changed it since
			// this call's own write, including an out-of-band update), there is nothing safe to restore
			// onto, so it's treated as a no-op.
			matched, restoreErr := dbadapter.AuthDBClient.RestfulAPIReplaceOneNoUpsert(authSubsDataColl, writtenFilter, previousAuth)
			if restoreErr != nil {
				logger.DbLog.Errorf("failed to restore authentication subscription for %s after amData create failure; manual reconciliation required: %+v", imsi, restoreErr)
			} else if !matched {
				logger.DbLog.Warnf("authentication subscription for %s changed since this call's write; skipping restore", imsi)
			}
		}
		return err
	}
	logger.WebUILog.Infof("successfully created authentication subscription in amData collection: %s", imsi)
	return nil
}

// amData isn't rewritten here: the caller has already confirmed the subscriber's amData record
// exists, so re-setting its ueId would be a no-op that only adds a second point of failure.
func subscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{ueIdKey: imsi}
	// Read the current document and condition the update on it still matching exactly: a blind
	// upsert here (the previous behavior) could silently overwrite a concurrent webconsole write, or
	// an out-of-band update such as a UDR sequence number re-synchronisation, that landed between
	// this read and the write below.
	currentAuth, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to read authentication subscription before update for %s: %+v", imsi, err)
		return err
	}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA[ueIdKey] = imsi
	if currentAuth == nil {
		// The caller already confirmed the subscriber (amData) exists, so this is not a "no such
		// subscriber" case: it's a credential missing from AuthDBClient entirely -- e.g. a subscriber
		// provisioned before issue #585 was fixed (auth writes landed on the wrong server), or an
		// amData-only orphan left by a prior partial create. There is nothing to condition a write on,
		// so repair it with a direct insert instead of aborting, same as create's previousAuth==nil
		// branch.
		if _, err = dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA); err != nil {
			logger.DbLog.Errorf("failed to repair missing authentication subscription for %s: %+v", imsi, err)
			return err
		}
		logger.WebUILog.Infof("created missing authentication subscription during update: %s", imsi)
		return nil
	}
	matched, err := dbadapter.AuthDBClient.RestfulAPIUpdateOneNoUpsert(authSubsDataColl, bson.M(currentAuth), authDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update authentication subscription error: %+v", err)
		return err
	}
	if !matched {
		err := fmt.Errorf("authentication subscription for %s changed concurrently; aborting update", imsi)
		logger.DbLog.Errorf("%v", err)
		return err
	}
	logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %s", imsi)
	return nil
}

// mergeBsonFields reconstructs the complete document RestfulAPIUpdateOneNoUpsert's $set-merge
// produces against a collection: base's fields with overlay's applied on top. Used to build a
// compare-and-swap filter against the full live document instead of just the subset a later
// write touched, so a change to ANY field -- not only the ones overlay carries -- is detected.
func mergeBsonFields(base map[string]any, overlay bson.M) bson.M {
	merged := bson.M{}
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	return merged
}

// amData is deleted first: if it fails, the credential is left untouched; if it succeeds but the
// credential delete below fails, the orphaned credential is harmless (amData is already gone, so
// recreation is allowed, and the next create's upsert overwrites the stale credential). Each
// delete is scoped to that collection's own current generation (read from its own client just
// before deleting), never derived from the other collection's snapshot: amData and the credential
// can independently already be gone or hold different generations (a prior partial delete, or
// unrelated cleanup such as device-group sync removing only one of the two), so cross-deriving one
// filter from the other's snapshot can leave an orphan unmatched and undeletable. Scoping still
// protects a concurrent recreate's newer credential and amData record from being removed instead.
//
// Also takes rwLock.RLock() for the same reason subscriberAuthenticationDataCreate does: so
// device-group/slice sync (which hold rwLock.Lock() across reading this credential and writing
// amData) can never interleave with this delete's own read-then-remove sequence.
func subscriberAuthenticationDataDelete(imsi string) error {
	rwLock.RLock()
	defer rwLock.RUnlock()

	filter := bson.M{ueIdKey: imsi}
	authSnapshot, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to read authentication subscription before delete: %+v", err)
		return err
	}
	amDataSnapshot, err := dbadapter.CommonDBClient.RestfulAPIGetOne(amDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to read amData before delete: %+v", err)
		return err
	}
	if err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, generationScopedFilter(imsi, amDataSnapshot)); err != nil {
		logger.DbLog.Errorf("failed to delete amData: %+v", err)
		return err
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %s", imsi)
	if err := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, generationScopedFilter(imsi, authSnapshot)); err != nil {
		logger.DbLog.Errorf("failed to delete authentication subscription; manual reconciliation required for %s: %+v", imsi, err)
		return err
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from authenticationSubscription collection: %v", imsi)
	return nil
}

// generationScopedFilter scopes a delete to the generation recorded in snapshot. A snapshot with
// no token -- legacy data predating the generation marker, or no document at all -- must only
// match other equally-untokened documents, never be treated as a wildcard that could remove a
// concurrently-created, tokened generation.
func generationScopedFilter(imsi string, snapshot map[string]any) bson.M {
	filter := bson.M{ueIdKey: imsi}
	if token, ok := snapshot[provisioningTokenKey]; ok {
		filter[provisioningTokenKey] = token
	} else {
		filter[provisioningTokenKey] = bson.M{"$exists": false}
	}
	return filter
}

func getDeletedImsisList(group, prevGroup *configmodels.DeviceGroups) (dimsis []string) {
	if prevGroup == nil {
		return
	}

	if group == nil {
		return prevGroup.Imsis
	}

	for _, pimsi := range prevGroup.Imsis {
		var found bool
		for _, imsi := range group.Imsis {
			if pimsi == imsi {
				found = true
			}
		}
		if !found {
			dimsis = append(dimsis, pimsi)
		}
	}
	return
}

func removeSubscriberEntriesRelatedToDeviceGroups(mcc, mnc, imsi string) error {
	filterImsiOnly := bson.M{ueIdKey: "imsi-" + imsi}
	filter := bson.M{ueIdKey: "imsi-" + imsi, servingPlmnIdKey: mcc + mnc}
	sessionRunner := dbadapter.GetSessionRunner(dbadapter.CommonDBClient)

	err := sessionRunner(context.TODO(), func(sc context.Context) error {
		// AM policy
		err := dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, amPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.DbLog.Errorf("failed to delete AM policy data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SM policy
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, smPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SM policy data for IMSI %s: %+v", imsi, err)
			return err
		}
		// AM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, amDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete AM data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, smDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SM data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SMF selection
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, smfSelDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SMF selection data for IMSI %s: %+v", imsi, err)
			return err
		}
		return nil
	})
	if err != nil {
		logger.DbLog.Errorf("failed to delete subscriber entries related to device groups for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to delete subscriber entries related to device groups for IMSI %s", imsi)
	return nil
}

func updateSubscriberInDeviceGroups(imsi string) (int, error) {
	filterByImsi := bson.M{
		"imsis": imsi,
	}
	rawDeviceGroups, err := dbadapter.CommonDBClient.RestfulAPIGetMany(devGroupDataColl, filterByImsi)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch device groups: %+v", err)
		return http.StatusInternalServerError, err
	}
	for _, rawDeviceGroup := range rawDeviceGroups {
		var deviceGroup configmodels.DeviceGroups
		if err = json.Unmarshal(configmodels.MapToByte(rawDeviceGroup), &deviceGroup); err != nil {
			logger.DbLog.Errorf("error unmarshaling device group: %+v", err)
			return http.StatusInternalServerError, err
		}
		// This write path bypasses deviceGroupPostHelper, so it must relabel the rate unit itself
		// rather than persist whatever unit a group written before that helper did carry.
		labelStoredDeviceGroupRatesAsBps(&deviceGroup)
		filteredImsis := []string{}
		for _, currImsi := range deviceGroup.Imsis {
			if currImsi != imsi {
				filteredImsis = append(filteredImsis, currImsi)
			}
		}
		deviceGroup.Imsis = filteredImsis
		prevDevGroup, err := getDeviceGroupByName(deviceGroup.DeviceGroupName)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to look up device group %s: %w", deviceGroup.DeviceGroupName, err)
		}
		if statusCode, err := handleDeviceGroupPost(&deviceGroup, prevDevGroup); err != nil {
			logger.ConfigLog.Errorf("error posting device group %+v: %+v", deviceGroup, err)
			return statusCode, err
		}
	}

	return http.StatusOK, nil
}
