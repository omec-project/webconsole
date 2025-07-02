// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

var (
	imsiDataLock sync.RWMutex
	ImsiData     = make(map[string]*models.AuthenticationSubscription)
)

func addSubscriber(imsi string, data *models.AuthenticationSubscription) {
	imsiDataLock.Lock()
	defer imsiDataLock.Unlock()
	ImsiData[imsi] = data
}

func removeSubscriber(imsi string) {
	imsiDataLock.Lock()
	defer imsiDataLock.Unlock()
	delete(ImsiData, imsi)
}

func getSubscriber(imsi string) *models.AuthenticationSubscription {
	imsiDataLock.RLock()
	defer imsiDataLock.RUnlock()
	data := ImsiData[imsi]
	return data
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
		logger.DbLog.Errorf("could not unmarshall subscriber %+v", authSubDataInterface)
		return
	}
	return authSubData
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	logger.WebUILog.Infof("%+v", authSubData)
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// write to AuthDB
	if _, err := dbadapter.AuthDBClient.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA); err != nil {
		logger.DbLog.Errorf("failed to update authentication subscription error: %+v", err)
		return err
	}
	logger.WebUILog.Infof("updated authentication subscription in authenticationSubscription collection: %s", imsi)
	// write to CommonDB
	basicAmData := map[string]interface{}{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson); err != nil {
		logger.DbLog.Errorf("failed to update amData error: %+v", err)
		// rollback AuthDB operation
		if cleanupErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(authSubsDataColl, filter); cleanupErr != nil {
			logger.DbLog.Errorf("rollback failed after authData op error: %+v", cleanupErr)
			return fmt.Errorf("authData update failed: %w, rollback failed: %+v", err, cleanupErr)
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Infof("successfully updated authentication subscription in amData collection: %s", imsi)
	return nil
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// get backup
	backup, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to get backup data for authentication subscription: %+v", err)
	}
	// write to AuthDB
	if _, err = dbadapter.AuthDBClient.RestfulAPIPutOne(authSubsDataColl, filter, authDataBsonA); err != nil {
		logger.DbLog.Errorf("failed to update authentication subscription error: %+v", err)
		return err
	}
	logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %s", imsi)
	// write to CommonDB
	basicAmData := map[string]interface{}{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err = dbadapter.CommonDBClient.RestfulAPIPutOne(amDataColl, filter, basicDataBson); err != nil {
		logger.DbLog.Errorf("failed to update amData error: %+v", err)
		// restore old auth data if any
		if backup != nil {
			_, err = dbadapter.AuthDBClient.RestfulAPIPutOne(authSubsDataColl, filter, backup)
			if err != nil {
				logger.DbLog.Errorf("failed to restore backup data for authentication subscription error: %+v", err)
			}
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully updated authentication subscription in amData collection: %s", imsi)
	return nil
}

func (subscriberAuthData DatabaseSubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) error {
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %s", imsi)
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
				logger.DbLog.Errorf("rollback failed after amData delete error error: %+v", restoreErr)
				return fmt.Errorf("amData delete failed: %w, rollback failed: %w", err, restoreErr)
			}
			return fmt.Errorf("amData delete failed, rolled back AuthDB change: %w", err)
		}
		return fmt.Errorf("amData delete failed, unable to rollback AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %s", imsi)
	return nil
}

type MemorySubscriberAuthenticationData struct {
	SubscriberAuthenticationData
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	return getSubscriber(imsi)
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	basicAmData := map[string]interface{}{
		"ueId": imsi,
	}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	logger.WebUILog.Debugf("insert/update authentication subscription in amData collection: %s", imsi)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, basicDataBson); err != nil {
		return fmt.Errorf("failed to update amData: %w", err)
	}
	logger.WebUILog.Debugf("successfully inserted/updated authentication subscription in amData collection: %s", imsi)
	addSubscriber(imsi, authSubData)
	logger.WebUILog.Debugf("insert/update authentication subscription in memory: %s", imsi)
	return nil
}

func (subscriberAuthData MemorySubscriberAuthenticationData) SubscriberAuthenticationDataDelete(imsi string) error {
	filter := bson.M{"ueId": imsi}
	if err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(amDataColl, filter); err != nil {
		return fmt.Errorf("failed to delete from amData collection: %w", err)
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %s", imsi)
	removeSubscriber(imsi)
	logger.WebUILog.Debugf("delete authentication subscription from memory: %s", imsi)
	return nil
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

func handleSubscriberDelete(imsi string) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	subscriberAuthData := DatabaseSubscriberAuthenticationData{}
	err := subscriberAuthData.SubscriberAuthenticationDataDelete(imsi)
	if err != nil {
		logger.DbLog.Errorln("SubscriberAuthDataDelete error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber delete for IMSI: %s", imsi)
	return nil
}

func handleSubscriberPut(imsi string, authSubData *models.AuthenticationSubscription) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	subscriberAuthData := DatabaseSubscriberAuthenticationData{}
	err := subscriberAuthData.SubscriberAuthenticationDataUpdate(imsi, authSubData)
	if err != nil {
		logger.DbLog.Errorln("Subscriber Authentication Data Update Error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber update for IMSI: %s", imsi)
	return nil
}

func handleSubscriberPost(imsi string, authSubData *models.AuthenticationSubscription) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	subscriberAuthData := DatabaseSubscriberAuthenticationData{}
	err := subscriberAuthData.SubscriberAuthenticationDataCreate(imsi, authSubData)
	if err != nil {
		logger.DbLog.Errorln("Subscriber Authentication Data Create Error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber post for IMSI: %s", imsi)
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
	var deviceGroupUpdateMessages []configmodels.ConfigMessage
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
		deviceGroupUpdateMessage := configmodels.ConfigMessage{
			MsgType:      configmodels.Device_group,
			MsgMethod:    configmodels.Post_op,
			DevGroupName: deviceGroup.DeviceGroupName,
			DevGroup:     &deviceGroup,
		}
		deviceGroupUpdateMessages = append(deviceGroupUpdateMessages, deviceGroupUpdateMessage)
	}
	for _, msg := range deviceGroupUpdateMessages {
		configChannel <- &msg
		logger.WebUILog.Infof("device group [%s] update sent to config channel", msg.DevGroupName)
	}
	return http.StatusOK, nil
}
