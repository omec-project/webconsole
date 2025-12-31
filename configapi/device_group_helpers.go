// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

var rwLock sync.RWMutex

const (
	KBPS = 1000
	MBPS = 1000000
	GBPS = 1000000000
)

func deviceGroupDeleteHelper(groupName string) error {
	logger.ConfigLog.Infof("received Delete Group %s request", groupName)
	if err := updateDeviceGroupInNetworkSlices(groupName); err != nil {
		return fmt.Errorf("error updating device group: %s in network slices: %+v", groupName, err)
	}
	if err := handleDeviceGroupDelete(groupName); err != nil {
		return fmt.Errorf("error deleting device group %s: %+v", groupName, err)
	}
	return nil
}

func updateDeviceGroupInNetworkSlices(groupName string) error {
	filterByDeviceGroup := bson.M{"site-device-group": groupName}
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filterByDeviceGroup)
	if err != nil {
		logger.AppLog.Errorf("failed to retrieve network slices error: %+v", err)
		return err
	}
	var errorOccurred bool
	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			logger.AppLog.Errorf("could not unmarshal network slice %s", rawNetworkSlice)
			errorOccurred = true
			continue
		}
		prevSlice := getSliceByName(networkSlice.SliceName)
		networkSlice.SiteDeviceGroup = slices.DeleteFunc(networkSlice.SiteDeviceGroup, func(existingDG string) bool {
			return groupName == existingDG
		})
		if statusCode, err := updateNS(networkSlice, *prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice: %s status code: %d error: %+v", networkSlice.SliceName, statusCode, err)
			errorOccurred = true
			continue
		}
	}
	if errorOccurred {
		return fmt.Errorf("one or more network slice updates failed (see logs)")
	}
	return nil
}

func deviceGroupPostHelper(requestDeviceGroup configmodels.DeviceGroups, groupName string) (int, error) {
	logger.ConfigLog.Infof("received device group: %s", groupName)

	ipdomain := &requestDeviceGroup.IpDomainExpanded
	logger.ConfigLog.Infof("imsis.size: %v, Imsis: %s", len(requestDeviceGroup.Imsis), requestDeviceGroup.Imsis)
	logger.ConfigLog.Infof("IP Domain Name: %s", requestDeviceGroup.IpDomainName)
	logger.ConfigLog.Infof("IP Domain details: %+v", ipdomain)
	logger.ConfigLog.Infof("dnn name: %s", ipdomain.Dnn)
	logger.ConfigLog.Infof("ue pool: %s", ipdomain.UeIpPool)
	logger.ConfigLog.Infof("dns Primary: %s", ipdomain.DnsPrimary)
	logger.ConfigLog.Infof("dns Secondary: %s", ipdomain.DnsSecondary)
	logger.ConfigLog.Infof("ip mtu: %v", ipdomain.Mtu)
	logger.ConfigLog.Infof("device Group Name: %s", groupName)

	if ipdomain.UeDnnQos != nil {
		ipdomain.UeDnnQos.DnnMbrDownlink = convertToBps(ipdomain.UeDnnQos.DnnMbrDownlink, ipdomain.UeDnnQos.BitrateUnit)
		if ipdomain.UeDnnQos.DnnMbrDownlink < 0 {
			ipdomain.UeDnnQos.DnnMbrDownlink = math.MaxInt64
		}
		logger.ConfigLog.Infof("MbrDownLink: %v", ipdomain.UeDnnQos.DnnMbrDownlink)
		ipdomain.UeDnnQos.DnnMbrUplink = convertToBps(ipdomain.UeDnnQos.DnnMbrUplink, ipdomain.UeDnnQos.BitrateUnit)
		if ipdomain.UeDnnQos.DnnMbrUplink < 0 {
			ipdomain.UeDnnQos.DnnMbrUplink = math.MaxInt64
		}
		logger.ConfigLog.Infof("MbrUpLink: %v", ipdomain.UeDnnQos.DnnMbrUplink)
	}

	prevDevGroup := getDeviceGroupByName(groupName)
	requestDeviceGroup.DeviceGroupName = groupName
	if prevDevGroup == nil {
		logger.ConfigLog.Infof("creating new device group %s", groupName)
		statusCode, err := createDG(&requestDeviceGroup)
		if err != nil {
			return statusCode, err
		}
	} else {
		statusCode, err := updateDG(&requestDeviceGroup, prevDevGroup)
		if err != nil {
			return statusCode, err
		}
	}

	return http.StatusOK, nil
}

func createDG(devGroup *configmodels.DeviceGroups) (int, error) {
	if statusCode, err := handleDeviceGroupPost(devGroup, nil); err != nil {
		logger.ConfigLog.Errorf("error creating device group %+v: %+v", devGroup, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func updateDG(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	if statusCode, err := handleDeviceGroupPost(devGroup, prevDevGroup); err != nil {
		logger.ConfigLog.Errorf("error updating device group %+v: %+v", devGroup, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func convertToBps(val int64, unit string) int64 {
	switch strings.ToLower(unit) {
	case "bps":
		return val
	case "kbps":
		return val * KBPS
	case "mbps":
		return val * MBPS
	case "gbps":
		return val * GBPS
	default:
		logger.ConfigLog.Warnf("unknown bitrate unit: %s, defaulting to bps", unit)
		return val
	}
}

func handleDeviceGroupPost(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	filter := bson.M{"group-name": devGroup.DeviceGroupName}
	devGroupDataBsonA := configmodels.ToBsonM(devGroup)
	result, err := dbadapter.CommonDBClient.RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to post device group data for %s: %+v", devGroup.DeviceGroupName, err)
		return http.StatusInternalServerError, err
	}
	logger.AppLog.Infof("DB operation result for device group %s: %v",
		devGroup.DeviceGroupName, result)
	statusCode, err := syncSubConcurrentlyInGroup(devGroup, prevDevGroup)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		return statusCode, err
	}
	logger.AppLog.Debugf("succeeded to post device group data for %s", devGroup.DeviceGroupName)
	return http.StatusOK, nil
}

func syncSubConcurrentlyInGroup(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	syncSliceStopMutex.Lock()
	if SyncSliceStop {
		syncSliceStopMutex.Unlock()
		return http.StatusServiceUnavailable, errors.New("error: the sync function is running")
	}
	SyncSliceStop = true
	syncSliceStopMutex.Unlock()

	go func() {
		defer func() {
			syncSliceStopMutex.Lock()
			SyncSliceStop = false
			syncSliceStopMutex.Unlock()
		}()

		_, err := syncDeviceGroupSubscriber(devGroup, prevDevGroup)
		if err != nil {
			logger.AppLog.Errorf("error syncing subscribers: %s", err)
		}
	}()

	return 0, nil // Retorno inmediato, operación en background
}

var syncDeviceGroupSubscriber = func(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	rwLock.Lock()
	defer rwLock.Unlock()
	slice := findSliceByDeviceGroup(devGroup.DeviceGroupName)
	if slice == nil {
		logger.WebUILog.Infof("Device group %s not associated with any slice — skipping sync", devGroup.DeviceGroupName)
		return http.StatusOK, nil
	}
	logger.WebUILog.Infof("Device group %s is part of slice %s", devGroup.DeviceGroupName, slice.SliceName)
	if slice.SliceId.Sst == "" {
		err := fmt.Errorf("missing SST in slice %s", slice.SliceName)
		logger.AppLog.Errorln(err)
		return http.StatusBadRequest, err
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.AppLog.Errorf("could not parse SST %s", slice.SliceId.Sst)
		return http.StatusBadRequest, err
	}
	snssai := &models.Snssai{
		Sd:  slice.SliceId.Sd,
		Sst: int32(sVal),
	}
	var errorOccured bool
	wg := sync.WaitGroup{}

	for _, imsi := range devGroup.Imsis {
		/* update all current IMSIs */
		if subscriberAuthenticationDataGet("imsi-"+imsi) != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dnn := devGroup.IpDomainExpanded.Dnn
				err := updatePolicyAndProvisionedData(
					imsi,
					slice.SiteInfo.Plmn.Mcc,
					slice.SiteInfo.Plmn.Mnc,
					snssai,
					dnn,
					devGroup.IpDomainExpanded.UeDnnQos,
				)
				if err != nil {
					logger.AppLog.Errorf("updatePolicyAndProvisionedData failed for IMSI %s: %+v", imsi, err)
					errorOccured = true
				}
			}()
		}
	}

	// delete IMSI's that are removed
	dimsis := getDeletedImsisList(devGroup, prevDevGroup)
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

func handleDeviceGroupDelete(groupName string) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	filter := bson.M{"group-name": groupName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(devGroupDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed to delete device group data for %s: %+v", groupName, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to device group data for %s", groupName)
	return nil
}

func getDeviceGroupByName(name string) *configmodels.DeviceGroups {
	filter := bson.M{"group-name": name}
	devGroupDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(devGroupDataColl, filter)
	if err != nil {
		logger.AppLog.Warnln(err)
		return nil
	}
	var devGroupData configmodels.DeviceGroups
	err = json.Unmarshal(configmodels.MapToByte(devGroupDataInterface), &devGroupData)
	if err != nil {
		logger.AppLog.Errorf("could not unmarshall device group %s", devGroupDataInterface)
		return nil
	}
	return &devGroupData
}

func findSliceByDeviceGroup(DevGroupName string) *configmodels.Slice {
	for _, slice := range getSlices() {
		for _, dgName := range slice.SiteDeviceGroup {
			if dgName == DevGroupName {
				logger.WebUILog.Infof("device Group [%s] is part of slice: %s", dgName, slice.SliceName)
				return slice
			}
		}
	}
	return nil
}
