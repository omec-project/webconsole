// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var rwLock sync.RWMutex

const (
	KBPS = 1000
	MBPS = 1000000
	GBPS = 1000000000

	bitrateUnitBps  = "bps"
	bitrateUnitKbps = "kbps"

	// maxDeviceGroupBitrateBps is the largest device group rate that can be served as configured.
	// ConvertToString names Gbps at most and its numeral has to fit the uint16 every consumer reads
	// it into, so a higher rate is rendered in bps -- which nas's Session-AMBR converter maps to
	// "unit not used" and whose numeral does not parse, leaving the UE with no rate at all. This is
	// deliberately not the MaxInt32 bound the slice rules use: those are per-flow rates in a signed
	// 32-bit field, while these are a session's aggregate in a 64-bit one.
	maxDeviceGroupBitrateBps = maxReadableBitRateNumeral * GBPS
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
		logger.DbLog.Errorf("failed to retrieve network slices error: %+v", err)
		return err
	}
	var errorOccurred bool
	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			logger.DbLog.Errorf("could not unmarshal network slice %s", rawNetworkSlice)
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

	for i := range requestDeviceGroup.IpDomainsExpanded {
		ipdomain := &requestDeviceGroup.IpDomainsExpanded[i]
		logger.ConfigLog.Infof("IP Domain details [%d]: %+v", i, ipdomain)
		logger.ConfigLog.Infof("DNN Name : %v", ipdomain.Dnn)
		logger.ConfigLog.Infof("UE Pool  : %v", ipdomain.UeIpPool)
		logger.ConfigLog.Infof("DNS Primary : %v", ipdomain.DnsPrimary)
		logger.ConfigLog.Infof("DNS Secondary : %v", ipdomain.DnsSecondary)
		logger.ConfigLog.Infof("IP MTU : %v", ipdomain.Mtu)
		if ipdomain.UeDnnQos != nil {
			if err := validateUeDnnQosBitrates(ipdomain.UeDnnQos, groupName, ipdomain.Dnn); err != nil {
				return http.StatusBadRequest, err
			}
			ipdomain.UeDnnQos.DnnMbrDownlink = convertToBps(ipdomain.UeDnnQos.DnnMbrDownlink, ipdomain.UeDnnQos.BitrateUnit)
			logger.ConfigLog.Infof("MBR DownLink : %v", ipdomain.UeDnnQos.DnnMbrDownlink)
			ipdomain.UeDnnQos.DnnMbrUplink = convertToBps(ipdomain.UeDnnQos.DnnMbrUplink, ipdomain.UeDnnQos.BitrateUnit)
			logger.ConfigLog.Infof("MBR UpLink : %v", ipdomain.UeDnnQos.DnnMbrUplink)

			// Both rates are bps from here on, so the unit has to say so. A GET returns the stored
			// group, and returning the operator's original unit beside a normalised value both
			// contradicts the field description and multiplies the rates again if that document is
			// posted back -- a thousandfold for a group configured in Kbps, with the operator having
			// changed nothing.
			ipdomain.UeDnnQos.BitrateUnit = bitrateUnitBps
		}
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

// A rate that isValidDeviceGroupBitrate rejects is one that cannot be served as configured, so the
// device group is refused rather than accepted with a rate the operator never asked for. Before
// this check the negative rates were clamped to math.MaxInt64, which turned the smallest rate an
// operator can express into the largest one the element can carry.
func validateUeDnnQosBitrates(qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, groupName, dnn string) error {
	rates := []struct {
		name  string
		value int64
	}{
		{"dnn-mbr-uplink", qos.DnnMbrUplink},
		{"dnn-mbr-downlink", qos.DnnMbrDownlink},
	}
	for _, rate := range rates {
		if !isValidDeviceGroupBitrate(rate.value, qos.BitrateUnit) {
			return fmt.Errorf("invalid %s %d %q for DNN %s in device group %s", rate.name, rate.value, qos.BitrateUnit, dnn, groupName)
		}
	}
	return nil
}

func convertToBps(val int64, unit string) int64 {
	multiplier, known := bitrateMultiplier(unit)
	if !known {
		logger.ConfigLog.Warnf("unknown bitrate unit: %s, defaulting to bps", unit)
	}
	return val * multiplier
}

// bitrateMultiplier is the bps factor for a unit, and whether the unit was recognised. It is
// separate from convertToBps so that validation, which needs the factor before anything is
// converted, does not log the unknown-unit warning a second time for the same rule.
func bitrateMultiplier(unit string) (int64, bool) {
	switch strings.ToLower(unit) {
	case bitrateUnitBps:
		return 1, true
	case bitrateUnitKbps:
		return KBPS, true
	case "mbps":
		return MBPS, true
	case "gbps":
		return GBPS, true
	}
	return 1, false
}

func handleDeviceGroupPost(devGroup *configmodels.DeviceGroups, prevDevGroup *configmodels.DeviceGroups) (int, error) {
	filter := bson.M{groupNameKey: devGroup.DeviceGroupName}
	devGroupDataBsonA := configmodels.ToBsonM(devGroup)
	result, err := dbadapter.CommonDBClient.RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to post device group data for %s: %+v", devGroup.DeviceGroupName, err)
		return http.StatusInternalServerError, err
	}
	logger.DbLog.Infof("DB operation result for device group %s: %v",
		devGroup.DeviceGroupName, result)

	statusCode, err := syncDeviceGroupSubscriber(devGroup, prevDevGroup)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		return statusCode, err
	}
	logger.DbLog.Debugf("succeeded to post device group data for %s", devGroup.DeviceGroupName)
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
		logger.DbLog.Errorln(err)
		return http.StatusBadRequest, err
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.DbLog.Errorf("could not parse SST %s", slice.SliceId.Sst)
		return http.StatusBadRequest, err
	}
	snssai := &models.Snssai{
		Sd:  openapi.PtrString(slice.SliceId.Sd),
		Sst: int32(sVal),
	}
	var errorOccured bool
	dnnMap := make(map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos)
	for _, ipDomain := range devGroup.IpDomainsExpanded {
		if ipDomain.UeDnnQos != nil {
			dnnMap[ipDomain.Dnn] = append(dnnMap[ipDomain.Dnn], *ipDomain.UeDnnQos)
		}
	}

	// Calculate the aggregatedQoS
	var allQosProfiles []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos
	for _, qosList := range dnnMap {
		allQosProfiles = append(allQosProfiles, qosList...)
	}

	aggregatedQoS := aggregateQoS(allQosProfiles)
	for i, imsi := range devGroup.Imsis {
		/* update all current IMSIs */
		if subscriberAuthenticationDataGet("imsi-"+imsi) != nil {
			var gpsi string
			if devGroup.Msisdns != nil && i < len(devGroup.Msisdns) {
				gpsi = devGroup.Msisdns[i]
			}
			err = updatePolicyAndProvisionedData(
				imsi,
				gpsi,
				snssai,
				dnnMap,
				slice.SiteInfo.Plmn.Mcc,
				slice.SiteInfo.Plmn.Mnc,
				aggregatedQoS,
			)
			if err != nil {
				logger.DbLog.Errorf("updatePolicyAndProvisionedData failed for IMSI %s: %+v", imsi, err)
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
	filter := bson.M{groupNameKey: groupName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(devGroupDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to delete device group data for %s: %+v", groupName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to device group data for %s", groupName)
	return nil
}

func getDeviceGroupByName(name string) *configmodels.DeviceGroups {
	filter := bson.M{groupNameKey: name}
	devGroupDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(devGroupDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return nil
	}
	var devGroupData configmodels.DeviceGroups
	err = json.Unmarshal(configmodels.MapToByte(devGroupDataInterface), &devGroupData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall device group %s", devGroupDataInterface)
		return nil
	}
	labelStoredDeviceGroupRatesAsBps(&devGroupData)
	return &devGroupData
}

// labelStoredDeviceGroupRatesAsBps makes a stored group's unit describe the rates stored beside it.
//
// The rates in an IP domain are normalised to bps when the group is written, and every consumer of
// the stored group reads them that way -- backend/nfconfig converts them to a string with no
// reference to the unit at all. Groups written before the ingest path stored the unit to match
// still carry the one the operator posted, so a GET would return a bps value labelled Kbps, and
// posting that document back would multiply the rates again. aggregateQoS compares these units
// across IP domains, so a stale one also makes it report an inconsistency that no longer exists.
//
// Rewriting the label on the way out rather than the rows in place keeps the read path honest
// without a migration, and a group written since the ingest path started storing the unit is
// already bps, so this leaves it alone.
func labelStoredDeviceGroupRatesAsBps(devGroup *configmodels.DeviceGroups) {
	for i := range devGroup.IpDomainsExpanded {
		if devGroup.IpDomainsExpanded[i].UeDnnQos != nil {
			devGroup.IpDomainsExpanded[i].UeDnnQos.BitrateUnit = bitrateUnitBps
		}
	}
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
