// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package server

import (
	"encoding/json"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type SubscriberAuthenticationData interface {
	SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription)
	SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription)
	SubscriberAuthenticationDataDelete(imsi string)
}

type DatabaseSubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	filter := bson.M{"ueId": imsi}
	authSubDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
	err = json.Unmarshal(configmodels.MapToByte(authSubDataInterface), &authSubData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall subscriber %v", authSubDataInterface)
		return
	}
	return authSubData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) {
	logger.WebUILog.Debugf("insert/update authentication subscription in authenticationSubscription collection: %v", imsi)
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	_, err := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
	logger.WebUILog.Debugf("insert/update authentication subscription in amData collection: %v", imsi)
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	_, err = dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) {
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %v", imsi)
	filter := bson.M{"ueId": imsi}
	err := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
	logger.WebUILog.Debugf("delete authentication subscription from amData collection: %v", imsi)
	err = dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
}

type MemorySubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	return imsiData[imsi]
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) {
	logger.WebUILog.Debugf("insert/update authentication subscription in memory: %v", imsi)
	imsiData[imsi] = authSubData
	filter := bson.M{"ueId": imsi}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	logger.WebUILog.Debugf("insert/update authentication subscription in amData collection: %v", imsi)
	basicDataBson := configmodels.ToBsonM(basicAmData)
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
	if err != nil {
		logger.DbLog.Warnln(err)
	}
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) {
	logger.WebUILog.Debugf("delete authentication subscription from amData collection: %v", imsi)
	filter := bson.M{"ueId": imsi}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return
	}
	logger.WebUILog.Debugf("delete authentication subscription from memory: %v", imsi)
	delete(imsiData, imsi)
}
