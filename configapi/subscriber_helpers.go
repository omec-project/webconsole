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

func subscriberAuthenticationDataGet(imsi string) (authSubData *models.AuthenticationSubscription) {
	filter := bson.M{"ueId": imsi}
	authSubDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(AuthSubsDataColl, filter)
	if err != nil {
		logger.AppLog.Errorln(err)
		return
	}
	err = json.Unmarshal(configmodels.MapToByte(authSubDataInterface), &authSubData)
	if err != nil {
		logger.AppLog.Errorf("could not unmarshall subscriber %+v", authSubDataInterface)
		return
	}
	return authSubData
}

func SubscriberAuthenticationDataCreate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	logger.WebUILog.Infof("%+v", authSubData)
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// write to AuthDB
	if _, err := dbadapter.AuthDBClient.RestfulAPIPost(AuthSubsDataColl, filter, authDataBsonA); err != nil {
		logger.AppLog.Errorf("failed to update authentication subscription error: %+v", err)
		return err
	}
	logger.WebUILog.Infof("updated authentication subscription in authenticationSubscription collection: %s", imsi)
	// write to CommonDB
	basicAmData := map[string]any{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err := dbadapter.CommonDBClient.RestfulAPIPost(AmDataColl, filter, basicDataBson); err != nil {
		logger.AppLog.Errorf("failed to update amData error: %+v", err)
		// rollback AuthDB operation
		if cleanupErr := dbadapter.AuthDBClient.RestfulAPIDeleteOne(AuthSubsDataColl, filter); cleanupErr != nil {
			logger.AppLog.Errorf("rollback failed after authData op error: %+v", cleanupErr)
			return fmt.Errorf("authData update failed: %w, rollback failed: %+v", err, cleanupErr)
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Infof("successfully updated authentication subscription in amData collection: %s", imsi)
	return nil
}

func SubscriberAuthenticationDataUpdate(imsi string, authSubData *models.AuthenticationSubscription) error {
	filter := bson.M{"ueId": imsi}
	authDataBsonA := configmodels.ToBsonM(authSubData)
	authDataBsonA["ueId"] = imsi
	// get backup
	backup, err := dbadapter.AuthDBClient.RestfulAPIGetOne(AuthSubsDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed to get backup data for authentication subscription: %+v", err)
	}
	// write to AuthDB
	if _, err = dbadapter.AuthDBClient.RestfulAPIPutOne(AuthSubsDataColl, filter, authDataBsonA); err != nil {
		logger.AppLog.Errorf("failed to update authentication subscription error: %+v", err)
		return err
	}
	logger.WebUILog.Debugf("updated authentication subscription in authenticationSubscription collection: %s", imsi)
	// write to CommonDB
	basicAmData := map[string]any{"ueId": imsi}
	basicDataBson := configmodels.ToBsonM(basicAmData)
	if _, err = dbadapter.CommonDBClient.RestfulAPIPutOne(AmDataColl, filter, basicDataBson); err != nil {
		logger.AppLog.Errorf("failed to update amData error: %+v", err)
		// restore old auth data if any
		if backup != nil {
			_, err = dbadapter.AuthDBClient.RestfulAPIPutOne(AuthSubsDataColl, filter, backup)
			if err != nil {
				logger.AppLog.Errorf("failed to restore backup data for authentication subscription error: %+v", err)
			}
		}
		return fmt.Errorf("authData update failed, rolled back AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully updated authentication subscription in amData collection: %s", imsi)
	return nil
}

func subscriberAuthenticationDataDelete(imsi string) error {
	logger.WebUILog.Debugf("delete authentication subscription from authenticationSubscription collection: %s", imsi)
	filter := bson.M{"ueId": imsi}

	origAuthData, getErr := dbadapter.AuthDBClient.RestfulAPIGetOne(AuthSubsDataColl, filter)
	if getErr != nil {
		logger.AppLog.Errorln("failed to fetch original AuthDB record before delete:", getErr)
		return getErr
	}

	// delete in AuthDB
	err := dbadapter.AuthDBClient.RestfulAPIDeleteOne(AuthSubsDataColl, filter)
	if err != nil {
		logger.AppLog.Errorln(err)
		return err
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from authenticationSubscription collection: %v", imsi)

	err = dbadapter.CommonDBClient.RestfulAPIDeleteOne(AmDataColl, filter)
	if err != nil {
		logger.AppLog.Errorln(err)
		// rollback AuthDB operation
		if origAuthData != nil {
			_, restoreErr := dbadapter.AuthDBClient.RestfulAPIPost(AuthSubsDataColl, filter, origAuthData)
			if restoreErr != nil {
				logger.AppLog.Errorf("rollback failed after amData delete error error: %+v", restoreErr)
				return fmt.Errorf("amData delete failed: %w, rollback failed: %w", err, restoreErr)
			}
			return fmt.Errorf("amData delete failed, rolled back AuthDB change: %w", err)
		}
		return fmt.Errorf("amData delete failed, unable to rollback AuthDB change: %w", err)
	}
	logger.WebUILog.Debugf("successfully deleted authentication subscription from amData collection: %s", imsi)
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
		err := dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, AmPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.AppLog.Errorf("failed to delete AM policy data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SM policy
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, SmPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.AppLog.Errorf("failed to delete SM policy data for IMSI %s: %+v", imsi, err)
			return err
		}
		// AM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, AmDataColl, filter)
		if err != nil {
			logger.AppLog.Errorf("failed to delete AM data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, SmDataColl, filter)
		if err != nil {
			logger.AppLog.Errorf("failed to delete SM data for IMSI %s: %+v", imsi, err)
			return err
		}
		// SMF selection
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, SmfSelDataColl, filter)
		if err != nil {
			logger.AppLog.Errorf("failed to delete SMF selection data for IMSI %s: %+v", imsi, err)
			return err
		}
		return nil
	})
	if err != nil {
		logger.AppLog.Errorf("failed to delete subscriber entries related to device groups for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to delete subscriber entries related to device groups for IMSI %s", imsi)
	return nil
}

func updateSubscriberInDeviceGroupsWhenDeleteSub(imsi string) (int, error) {
	filterByImsi := bson.M{
		"imsis": imsi,
	}
	rawDeviceGroups, err := dbadapter.CommonDBClient.RestfulAPIGetMany(devGroupDataColl, filterByImsi)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch device groups: %+v", err)
		return http.StatusInternalServerError, err
	}
	for _, rawDeviceGroup := range rawDeviceGroups {
		var deviceGroup configmodels.DeviceGroups
		if err = json.Unmarshal(configmodels.MapToByte(rawDeviceGroup), &deviceGroup); err != nil {
			logger.AppLog.Errorf("error unmarshaling device group: %+v", err)
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

		filter := bson.M{"group-name": deviceGroup.DeviceGroupName}
		devGroupDataBsonA := configmodels.ToBsonM(deviceGroup)
		result, err := dbadapter.CommonDBClient.RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
		if err != nil {
			logger.AppLog.Errorf("failed to post device group data for %s: %+v", deviceGroup.DeviceGroupName, err)
			return http.StatusInternalServerError, err
		}
		logger.AppLog.Infof("DB operation result for device group %s: %v",
			deviceGroup.DeviceGroupName, result)

		slice := findSliceByDeviceGroup(deviceGroup.DeviceGroupName)
		if slice == nil {
			logger.WebUILog.Infof("Device group %s not associated with any slice — skipping sync", deviceGroup.DeviceGroupName)
			return http.StatusOK, nil
		}
		logger.WebUILog.Infof("Device group %s is part of slice %s", deviceGroup.DeviceGroupName, slice.SliceName)
		if slice.SliceId.Sst == "" {
			err := fmt.Errorf("missing SST in slice %s", slice.SliceName)
			logger.AppLog.Errorln(err)
			return http.StatusBadRequest, err
		}

		var errorOccured bool
		wg := sync.WaitGroup{}

		// delete IMSI's that are removed
		dimsis := getDeletedImsisList(&deviceGroup, prevDevGroup)
		for _, imsi := range dimsis {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := removeSubscriberEntriesRelatedToDeviceGroups(slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, imsi)
				if err != nil {
					logger.ConfigLog.Errorln(err)
					errorOccured = true
				}
			}()
		}
		wg.Wait()

		if errorOccured {
			return http.StatusInternalServerError, fmt.Errorf("syncDeviceGroupSubscriber failed, please check logs")
		} else {
			return http.StatusOK, nil
		}
	}

	return http.StatusOK, nil
}
