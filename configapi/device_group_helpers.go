// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	rwLock             sync.RWMutex
	subscriberAuthData SubscriberAuthenticationData
)

func handleDeviceGroupPost(devGroup configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) error {
	if devGroup.DeviceGroupName == "" {
		err := fmt.Errorf("device group name is empty")
		logger.DbLog.Errorw("device group name is required for posting device group data", "error", err)
		return err
	}

	filter := bson.M{"group-name": devGroup.DeviceGroupName}
	devGroupDataBsonA := configmodels.ToBsonM(devGroup)
	result, err := dbadapter.CommonDBClient.RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
	if err != nil {
		logger.DbLog.Errorw("failed to post device group data for %v: %v", devGroup.DeviceGroupName, err)
		return err
	}
	logger.DbLog.Infof("DB operation result for device group %s: %v",
		devGroup.DeviceGroupName, result)

	err = syncDeviceGroupSubscriber(devGroup, prevDevGroup)
	if err != nil {
		logger.WebUILog.Error(err.Error())
		return err
	}
	logger.DbLog.Debugf("succeeded to post device group data for %v", devGroup.DeviceGroupName)
	return nil
}

func syncDeviceGroupSubscriber(devGroup configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	slice := isDeviceGroupExistInSlice(devGroup.DeviceGroupName)
	if slice == nil {
		return nil
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.DbLog.Errorf("could not parse SST %v", slice.SliceId.Sst)
		return err
	}
	snssai := &models.Snssai{
		Sd:  slice.SliceId.Sd,
		Sst: int32(sVal),
	}
	mongoClient := dbadapter.CommonDBClient.(*mongoapi.MongoClient)
	sessRunner := dbadapter.RealSessionRunner(mongoClient.Client)

	var errorOccured bool
	for _, imsi := range devGroup.Imsis {
		/* update all current IMSIs */
		if subscriberAuthData.SubscriberAuthenticationDataGet("imsi-"+imsi) != nil {
			dnn := devGroup.IpDomainExpanded.Dnn
			err = updatePolicyAndProvisionedData(
				imsi,
				slice.SiteInfo.Plmn.Mcc,
				slice.SiteInfo.Plmn.Mnc,
				snssai,
				dnn,
				devGroup.IpDomainExpanded.UeDnnQos,
			)
			if err != nil {
				logger.DbLog.Errorf("updatePolicyAndProvisionedData failed for IMSI %s: %v", imsi, err)
				errorOccured = true
			}
		}
	}
	// delete IMSI's that are removed
	dimsis := getDeletedImsisList(&devGroup, prevDevGroup)
	for _, imsi := range dimsis {
		err = removeSubscriberEntriesRelatedToDeviceGroups(slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, imsi, sessRunner)
		if err != nil {
			logger.ConfigLog.Errorln(err)
			errorOccured = true
		}
	}

	if errorOccured {
		return fmt.Errorf("syncDeviceGroupSubscriber failed, please check logs")
	} else {
		return nil
	}
}

func handleDeviceGroupDelete(groupName string) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	filter := bson.M{"group-name": groupName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(devGroupDataColl, filter)
	if err != nil {
		logger.DbLog.Errorw("failed to delete device group data for %v: %v", groupName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to device group data for %v", groupName)
	return nil
}

func getDeviceGroupByName(name string) *configmodels.DeviceGroups {
	filter := bson.M{"group-name": name}
	devGroupDataInterface, errGetOne := dbadapter.CommonDBClient.RestfulAPIGetOne(devGroupDataColl, filter)
	if errGetOne != nil {
		logger.DbLog.Warnln(errGetOne)
	}
	var devGroupData configmodels.DeviceGroups
	err := json.Unmarshal(configmodels.MapToByte(devGroupDataInterface), &devGroupData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall device group %v", devGroupDataInterface)
	}
	return &devGroupData
}

func isDeviceGroupExistInSlice(DevGroupName string) *configmodels.Slice {
	for name, slice := range getSlices() {
		for _, dgName := range slice.SiteDeviceGroup {
			if dgName == DevGroupName {
				logger.WebUILog.Infof("device Group [%v] is part of slice: %v", dgName, name)
				return slice
			}
		}
	}
	return nil
}
