// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type SubscriberAuthenticationData interface {
	SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription)
	SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error
	SubscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error
	SubscriberAuthenticationDataDelete(imsi string) error
}

type DatabaseSubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

var imsiData map[string]*models.AuthenticationSubscription

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

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	if authSubData == nil {
		return fmt.Errorf("authentication subscription data is nil")
	}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// write to AuthDB
	if _, err := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA); err != nil {
		logger.DbLog.Errorw("failed to update authentication subscription", "error", err)
		return err
	}
	logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %v", imsi)
	// write to CommonDB
	basicAmData := map[string]interface{}{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson); err != nil {
		logger.DbLog.Errorw("failed to update amData", "error", err)
		// rollback AuthDB operation
		if cleanupErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter); cleanupErr != nil {
			logger.DbLog.Errorw("rollback failed after authData op", "error", cleanupErr)
			return fmt.Errorf("authData update failed: %v, rollback failed: %w", err, cleanupErr)
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully updated authentication subscription in amData collection: %v", imsi)
	return nil
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// write to AuthDB
	if _, err := dbadapter.AuthDBClient.RestfulAPIPutOne(authSubsDataColl, filter, authDataBsonA); err != nil {
		logger.DbLog.Errorw("failed to update authentication subscription", "error", err)
		return err
	}
	logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %v", imsi)
	// write to CommonDB
	basicAmData := map[string]interface{}{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPutOne(amDataColl, filter, basicDataBson); err != nil {
		logger.DbLog.Errorw("failed to update amData", "error", err)
		// rollback AuthDB operation
		if cleanupErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter); cleanupErr != nil {
			logger.DbLog.Errorw("rollback failed after authData op", "error", cleanupErr)
			return fmt.Errorf("authData update failed: %v, rollback failed: %w", err, cleanupErr)
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully updated authentication subscription in amData collection: %v", imsi)
	return nil
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) error {
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %v", imsi)
	filter := bson.M{"ueId": imsi}

	origAuthData, getErr := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if getErr != nil {
		logger.DbLog.Errorln("failed to fetch original AuthDB record before delete:", getErr)
		return getErr
	}

	// delete in AuthDB
	err := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		return err
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from authenticationSubscription collection: %v", imsi)

	err = dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		// rollback AuthDB operation
		if origAuthData != nil {
			_, restoreErr := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, origAuthData)
			if restoreErr != nil {
				logger.DbLog.Errorw("rollback failed after amData delete error", "error", restoreErr)
				return fmt.Errorf("amData delete failed: %v, rollback failed: %w", err, restoreErr)
			}
			return fmt.Errorf("amData delete failed, rolled back AuthDB change: %w", err)
		}
		return fmt.Errorf("amData delete failed, unable to rollback AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %v", imsi)
	return nil
}

type MemorySubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	return imsiData[imsi]
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	logger.WebUILog.Debugf("insert/update authentication subscription in amData collection: %v", imsi)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson); err != nil {
		return fmt.Errorf("failed to update amData: %w", err)
	}
	logger.WebUILog.Debugf("successfully inserted/updated authentication subscription in amData collection: %v", imsi)
	logger.WebUILog.Debugf("insert/update authentication subscription in memory: %v", imsi)
	imsiData[imsi] = authSubData
	return nil
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) error {
	filter := bson.M{"ueId": imsi}
	if err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter); err != nil {
		return fmt.Errorf("failed to delete from amData collection: %w", err)
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %v", imsi)
	logger.WebUILog.Debugf("delete authentication subscription from memory: %v", imsi)
	delete(imsiData, imsi)
	return nil
}
