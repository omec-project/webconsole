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
	authSubDataInterface, errGetOne := dbadapter.CommonDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if errGetOne != nil {
		logger.DbLog.Warnln(errGetOne)
	}
	err := json.Unmarshal(configmodels.MapToByte(authSubDataInterface), &authSubData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall subscriber %v", authSubDataInterface)
	}
	return authSubData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) {
	logger.WebUILog.Debugln("insert/update AuthenticationSubscription ", imsi)
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	_, errPost := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
	if errPost != nil {
		logger.DbLog.Warnln(errPost)
	}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	_, errPost = dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
	if errPost != nil {
		logger.DbLog.Warnln(errPost)
	}
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) {
	logger.WebUILog.Debugln("delete AuthenticationSubscription", imsi)
	filter := bson.M{"ueId": imsi}
	errDelAuthSubsDataColl := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter)
	if errDelAuthSubsDataColl != nil {
		logger.DbLog.Warnln(errDelAuthSubsDataColl)
	}
	errDelAmDataColl := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
	if errDelAmDataColl != nil {
		logger.DbLog.Warnln(errDelAmDataColl)
	}
}

type MemorySubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	return imsiData[imsi]
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) {
	logger.WebUILog.Debugln("insert/update AuthenticationSubscription ", imsi)
	imsiData[imsi] = authSubData
	filter := bson.M{"ueId": imsi}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	_, errPost := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson)
	if errPost != nil {
		logger.DbLog.Warnln(errPost)
	}
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) {
	logger.WebUILog.Debugln("delete AuthenticationSubscription ", imsi)
	filter := bson.M{"ueId": imsi}
	errDelAmDataColl := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter)
	if errDelAmDataColl != nil {
		logger.DbLog.Warnln(errDelAmDataColl)
	}
	logger.WebUILog.Debugln("remove imsi from memory", imsi)
	delete(imsiData, imsi)
}
