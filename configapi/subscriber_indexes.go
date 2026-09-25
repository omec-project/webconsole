// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/dbadapter"
)

type collectionIndex struct {
	collName string
	spec     mongoapi.IndexSpec
}

// commonDBSubscriberIndexes cover the per-subscriber collections webconsole
// writes. The UDR serves them to the UDM and PCF, and webconsole's own
// upserts, deletes and subscriber API filter them on the same keys, so each
// index serves the writer as well as the readers. Webconsole is their only
// writer, so it is the one that indexes them.
//
// Keys follow the readers' filters: provisioned data is partitioned by serving
// network and read by (ueId, servingPlmnId), policy data by ueId alone. None is
// unique: uniqueness is not needed for a lookup to seek, and a unique index
// over a collection that already holds a duplicate could never be created,
// which would stop webconsole from starting.
var commonDBSubscriberIndexes = []collectionIndex{
	{amDataColl, mongoapi.IndexSpec{
		Name: "amDataByUeIdAndServingPlmnId",
		Keys: mongoapi.AscendingKeys(ueIdKey, servingPlmnIdKey),
	}},
	{smDataColl, mongoapi.IndexSpec{
		Name: "smDataByUeIdAndServingPlmnId",
		Keys: mongoapi.AscendingKeys(ueIdKey, servingPlmnIdKey),
	}},
	{smfSelDataColl, mongoapi.IndexSpec{
		Name: "smfSelectionSubscriptionDataByUeIdAndServingPlmnId",
		Keys: mongoapi.AscendingKeys(ueIdKey, servingPlmnIdKey),
	}},
	{amPolicyDataColl, mongoapi.IndexSpec{
		Name: "amPolicyDataByUeId",
		Keys: mongoapi.AscendingKeys(ueIdKey),
	}},
	{smPolicyDataColl, mongoapi.IndexSpec{
		Name: "smPolicyDataByUeId",
		Keys: mongoapi.AscendingKeys(ueIdKey),
	}},
}

// authDBSubscriberIndexes are created through AuthDBClient, because that is the
// connection every reader of the credentials uses, here and in the UDR.
var authDBSubscriberIndexes = []collectionIndex{
	{authSubsDataColl, mongoapi.IndexSpec{
		Name: "authenticationSubscriptionByUeId",
		Keys: mongoapi.AscendingKeys(ueIdKey),
	}},
}

// EnsureSubscriberIndexes makes the subscriber collections carry their indexes,
// and returns the first index it could not ensure.
func EnsureSubscriberIndexes() error {
	for _, index := range commonDBSubscriberIndexes {
		if err := dbadapter.EnsureIndex(dbadapter.CommonDBClient, index.collName, index.spec); err != nil {
			return err
		}
		logger.InitLog.Infow("index is present", "collection", index.collName, "index", index.spec.Name)
	}
	for _, index := range authDBSubscriberIndexes {
		if err := dbadapter.EnsureIndex(dbadapter.AuthDBClient, index.collName, index.spec); err != nil {
			return err
		}
		logger.InitLog.Infow("index is present", "collection", index.collName, "index", index.spec.Name)
	}
	return nil
}
