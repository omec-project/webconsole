// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func subscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	filter := bson.M{"ueId": imsi}
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

func subscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	logger.WebUILog.Infof("%+v", authSubData)
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	basicAmData := map[string]any{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	authDbName := factory.WebUIConfig.Configuration.Mongodb.AuthKeysDbName
	sessionRunner := dbadapter.GetSessionRunner(dbadapter.CommonDBClient)
	return sessionRunner(context.TODO(), func(sc mongo.SessionContext) error {
		if _, err := dbadapter.CommonDBClient.RestfulAPIPostOnDB(sc, authDbName, authSubsDataColl, filter, authDataBsonA); err != nil {
			logger.DbLog.Errorf("failed to create authentication subscription error: %+v", err)
			return err
		}
		logger.WebUILog.Infof("created authentication subscription in authenticationSubscription collection: %s", imsi)
		if _, err := dbadapter.CommonDBClient.RestfulAPIPostWithContext(sc, amDataColl, filter, basicDataBson); err != nil {
			logger.DbLog.Errorf("failed to create amData error: %+v", err)
			return err
		}
		logger.WebUILog.Infof("successfully created authentication subscription in amData collection: %s", imsi)
		return nil
	})
}

func subscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	basicAmData := map[string]any{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	authDbName := factory.WebUIConfig.Configuration.Mongodb.AuthKeysDbName
	sessionRunner := dbadapter.GetSessionRunner(dbadapter.CommonDBClient)
	return sessionRunner(context.TODO(), func(sc mongo.SessionContext) error {
		if _, err := dbadapter.CommonDBClient.RestfulAPIPutOneOnDB(sc, authDbName, authSubsDataColl, filter, authDataBsonA); err != nil {
			logger.DbLog.Errorf("failed to update authentication subscription error: %+v", err)
			return err
		}
		logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %s", imsi)
		if _, err := dbadapter.CommonDBClient.RestfulAPIPutOneWithContext(sc, amDataColl, filter, basicDataBson); err != nil {
			logger.DbLog.Errorf("failed to update amData error: %+v", err)
			return err
		}
		logger.WebUILog.Debugf("successfully updated authentication subscription in amData collection: %s", imsi)
		return nil
	})
}

func subscriberAuthenticationDataDelete(imsi string) error {
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %s", imsi)
	filter := bson.M{"ueId": imsi}
	authDbName := factory.WebUIConfig.Configuration.Mongodb.AuthKeysDbName
	sessionRunner := dbadapter.GetSessionRunner(dbadapter.CommonDBClient)
	return sessionRunner(context.TODO(), func(sc mongo.SessionContext) error {
		if err := dbadapter.CommonDBClient.RestfulAPIDeleteOneOnDB(sc, authDbName, authSubsDataColl, filter); err != nil {
			logger.DbLog.Errorf("failed to delete authentication subscription: %+v", err)
			return err
		}
		logger.WebUILog.Debugf("successfully deleted authentication subscription from authenticationSubscription collection: %v", imsi)
		if err := dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, amDataColl, filter); err != nil {
			logger.DbLog.Errorf("failed to delete amData: %+v", err)
			return err
		}
		logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %s", imsi)
		return nil
	})
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
	filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	sessionRunner := dbadapter.GetSessionRunner(dbadapter.CommonDBClient)

	err := sessionRunner(context.TODO(), func(sc mongo.SessionContext) error {
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
		filteredImsis := []string{}
		for _, currImsi := range deviceGroup.Imsis {
			if currImsi != imsi {
				filteredImsis = append(filteredImsis, currImsi)
			}
		}
		deviceGroup.Imsis = filteredImsis
		prevDevGroup := getDeviceGroupByName(deviceGroup.DeviceGroupName)
		if statusCode, err := handleDeviceGroupPost(&deviceGroup, prevDevGroup); err != nil {
			logger.ConfigLog.Errorf("error posting device group %+v: %+v", deviceGroup, err)
			return statusCode, err
		}
	}

	return http.StatusOK, nil
}
