// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
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
	KPS = 1000
	MPS = 1000000
	GPS = 1000000000
)

func deviceGroupDeleteHelper(groupName string) error {
	logger.ConfigLog.Infof("received Delete Group %v request", groupName)
	if err := updateDeviceGroupInNetworkSlices(groupName); err != nil {
		return fmt.Errorf("error updating device group: %v in network slices: %v", groupName, err)
	}
	if err := handleDeviceGroupDelete(groupName); err != nil {
		return fmt.Errorf("error deleting device group %v: %v", groupName, err)
	}
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = configmodels.Delete_op
	msg.DevGroupName = groupName
	configChannel <- &msg

	logger.ConfigLog.Infof("successfully Added Device Group [%v] with delete_op to config channel", groupName)
	return nil
}

func updateDeviceGroupInNetworkSlices(groupName string) error {
	filterByDeviceGroup := bson.M{"site-device-group": groupName}
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filterByDeviceGroup)
	if err != nil {
		logger.DbLog.Errorw("failed to retrieve network slices", "error", err)
		return err
	}
	var errorOccurred bool
	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			logger.DbLog.Errorf("could not unmarshal network slice %v", rawNetworkSlice)
			errorOccurred = true
			continue
		}
		prevSlice := getSliceByName(networkSlice.SliceName)
		networkSlice.SiteDeviceGroup = slices.DeleteFunc(networkSlice.SiteDeviceGroup, func(existingDG string) bool {
			return groupName == existingDG
		})
		if statusCode, err := updateNS(&networkSlice, prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice: %v status code: %v error: %v", networkSlice.SliceName, statusCode, err)
			errorOccurred = true
			continue
		}

		msg := &configmodels.ConfigMessage{
			MsgMethod: configmodels.Post_op,
			MsgType:   configmodels.Network_slice,
			Slice:     &networkSlice,
			SliceName: networkSlice.SliceName,
		}
		configChannel <- msg
		logger.ConfigLog.Infof("network slice [%v] update sent to config channel", networkSlice.SliceName)
	}
	if errorOccurred {
		return fmt.Errorf("one or more network slice updates failed (see logs)")
	}
	return nil
}

func deviceGroupPostHelper(requestDeviceGroup configmodels.DeviceGroups, msgOp int, groupName string) (int, error) {
	logger.ConfigLog.Infof("received device group: %v", groupName)

	ipdomain := &requestDeviceGroup.IpDomainExpanded
	logger.ConfigLog.Infof("imsis.size: %v, Imsis: %v", len(requestDeviceGroup.Imsis), requestDeviceGroup.Imsis)
	logger.ConfigLog.Infof("IP Domain Name: %v", requestDeviceGroup.IpDomainName)
	logger.ConfigLog.Infof("IP Domain details: %v", ipdomain)
	logger.ConfigLog.Infof("dnn name: %v", ipdomain.Dnn)
	logger.ConfigLog.Infof("ue pool: %v", ipdomain.UeIpPool)
	logger.ConfigLog.Infof("dns Primary: %v", ipdomain.DnsPrimary)
	logger.ConfigLog.Infof("dns Secondary: %v", ipdomain.DnsSecondary)
	logger.ConfigLog.Infof("ip mtu: %v", ipdomain.Mtu)
	logger.ConfigLog.Infof("device Group Name: %v", groupName)

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
		logger.ConfigLog.Infof("creating new device group %v", groupName)
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

	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = msgOp
	msg.DevGroup = &requestDeviceGroup
	msg.DevGroupName = groupName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully added Device Group [%v] to config channel", groupName)
	return http.StatusOK, nil
}

func createDG(devGroup *configmodels.DeviceGroups) (int, error) {
	if statusCode, err := handleDeviceGroupPost(devGroup, nil); err != nil {
		logger.ConfigLog.Errorf("error creating device group %v: %v", devGroup, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func updateDG(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	if statusCode, err := handleDeviceGroupPost(devGroup, prevDevGroup); err != nil {
		logger.ConfigLog.Errorf("error updating device group %v: %v", devGroup, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func convertToBps(val int64, unit string) int64 {
	switch strings.ToLower(unit) {
	case "bps":
		return val
	case "kbps":
		return val * KPS
	case "mbps":
		return val * MPS
	case "gbps":
		return val * GPS
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
		logger.DbLog.Errorw("failed to post device group data for %v: %v", devGroup.DeviceGroupName, err)
		return http.StatusInternalServerError, err
	}
	logger.DbLog.Infof("DB operation result for device group %s: %v",
		devGroup.DeviceGroupName, result)

	statusCode, err := syncDeviceGroupSubscriber(devGroup, prevDevGroup)
	if err != nil {
		logger.WebUILog.Error(err.Error())
		return statusCode, err
	}
	logger.DbLog.Debugf("succeeded to post device group data for %v", devGroup.DeviceGroupName)
	return http.StatusOK, nil
}

func syncDeviceGroupSubscriber(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
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
		logger.DbLog.Error(err)
		return http.StatusBadRequest, err
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.DbLog.Errorf("could not parse SST %v", slice.SliceId.Sst)
		return http.StatusBadRequest, err
	}
	snssai := &models.Snssai{
		Sd:  slice.SliceId.Sd,
		Sst: int32(sVal),
	}
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
	dimsis := getDeletedImsisList(devGroup, prevDevGroup)
	for _, imsi := range dimsis {
		err = removeSubscriberEntriesRelatedToDeviceGroups(slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, imsi)
		if err != nil {
			logger.ConfigLog.Errorln(err)
			errorOccured = true
		}
	}

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
		logger.DbLog.Errorw("failed to delete device group data for %v: %v", groupName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to device group data for %v", groupName)
	return nil
}

func getDeviceGroupByName(name string) *configmodels.DeviceGroups {
	filter := bson.M{"group-name": name}
	devGroupDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(devGroupDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return nil
	}
	var devGroupData configmodels.DeviceGroups
	err = json.Unmarshal(configmodels.MapToByte(devGroupDataInterface), &devGroupData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall device group %v", devGroupDataInterface)
		return nil
	}
	return &devGroupData
}

func findSliceByDeviceGroup(DevGroupName string) *configmodels.Slice {
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
