// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SubscriberAuthenticationData interface {
	SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription)
	SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription, sessionRunner dbadapter.SessionRunner) error
	SubscriberAuthenticationDataDelete(imsi string, sessionRunner dbadapter.SessionRunner) error
}

type DatabaseSubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	filter := bson.M{"ueId": imsi}
	authSubDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		return
	}
	err = json.Unmarshal(configmodels.MapToByte(authSubDataInterface), &authSubData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall subscriber %v", authSubDataInterface)
		return
	}
	return authSubData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription, sessionRunner dbadapter.SessionRunner) error {
	ctx := context.TODO()
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi

	_, err := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
	if err != nil {
		logger.DbLog.Errorw("failed to update authentication subscription", "error", err)
		return err
	}
	basicAmData := map[string]interface{}{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	err = sessionRunner(ctx, func(sc mongo.SessionContext) error {
		_, dbErr := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
		if dbErr != nil {
			cleanupErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter)
			if cleanupErr != nil {
				logger.DbLog.Errorw("rollback failed after amData op", "error", cleanupErr)
				return fmt.Errorf("amData update failed: %v, rollback failed: %w", err, cleanupErr)
			}
			return fmt.Errorf("amData update failed, rolled back AuthDB change: %w", err)
		}
		return nil
	})
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}

	return nil
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string, sessionRunner dbadapter.SessionRunner) error {
	filter := bson.M{"ueId": imsi}
	ctx := context.TODO()
	oldAuthRecord, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln("failed to fetch record for potential compensation:", err)
		return err
	}
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %v", imsi)
	err = dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}

	logger.WebUILog.Debugf("delete authentication subscription from amData collection: %v", imsi)
	err = sessionRunner(ctx, func(sc mongo.SessionContext) error {
		dbErr := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
		if dbErr != nil {
			// restore AuthDB record
			logger.DbLog.Errorw("failed to delete from CommonDB; attempting to restore AuthDB", "error", dbErr)
			_, restoreErr := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, oldAuthRecord)
			if restoreErr != nil {
				logger.DbLog.Errorw("compensation (restore) failed after CommonDB delete error", "error", restoreErr)
				return fmt.Errorf("CommonDB delete error: %v, compensation error: %w", err, restoreErr)
			}
			return fmt.Errorf("commonDB delete error, compensated by restoring AuthDB: %w", err)
		}

		return nil
	})
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}
	return nil
}

type MemorySubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	return imsiData[imsi]
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription, sessionRunner dbadapter.SessionRunner) error {
	filter := bson.M{"ueId": imsi}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)

	ctx := context.TODO()
	err := sessionRunner(ctx, func(sc mongo.SessionContext) error {
		logger.WebUILog.Debugf("insert/update authentication subscription in amData collection: %v", imsi)
		_, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
		if err != nil {
			return fmt.Errorf("failed to update amData: %w", err)
		}
		return nil
	})
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}
	logger.WebUILog.Debugf("insert/update authentication subscription in memory: %v", imsi)
	imsiData[imsi] = authSubData
	return nil
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string, sessionRunner dbadapter.SessionRunner) error {
	filter := bson.M{"ueId": imsi}
	ctx := context.TODO()
	err := sessionRunner(ctx, func(sc mongo.SessionContext) error {
		err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
		if err != nil {
			return fmt.Errorf("failed to delete from amData collection: %w", err)
		}
		return nil
	})
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}
	logger.WebUILog.Debugf("delete authentication subscription from memory: %v", imsi)
	delete(imsiData, imsi)
	return nil
}
