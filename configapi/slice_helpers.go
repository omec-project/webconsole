// Copyright (c) 2026 Intel Corporation
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var execCommand = exec.Command

func networkSliceDeleteHelper(sliceName string) error {
	if err := handleNetworkSliceDelete(sliceName); err != nil {
		logger.ConfigLog.Errorf("Error deleting slice %s: %+v", sliceName, err)
		return err
	}
	return nil
}

func networkSlicePostHelper(c *gin.Context, sliceName string) (int, error) {
	logger.ConfigLog.Infof("received slice: %s", sliceName)
	requestSlice, statusCode, err := parseAndValidateSliceRequest(c, sliceName)
	if err != nil {
		return statusCode, err
	}

	logSliceMetadata(requestSlice)
	normalizeApplicationFilteringRules(&requestSlice)
	requestSlice.SliceName = sliceName
	prevSlice, err := getSliceByName(sliceName)
	if err != nil {
		// An inconclusive lookup must not be treated as "slice does not exist": that would take the
		// create path below, which can silently overwrite an existing slice's PLMN and reintroduce
		// the duplicate-subscriber bug this check exists to prevent.
		return http.StatusInternalServerError, fmt.Errorf("failed to look up existing network slice %s: %w", sliceName, err)
	}

	if prevSlice == nil {
		logger.ConfigLog.Infof("Adding new slice [%s]", sliceName)
		if statusCode, err := createNS(requestSlice); err != nil {
			logger.ConfigLog.Errorf("Error creating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	} else {
		// A zero-value previous PLMN with no device groups attached could never have had a
		// subscriber synced under it (syncSubscribersOnSliceCreateOrUpdate only writes records for
		// a slice's device groups), so assigning a real PLMN in that case is not a change to
		// reject. Once device groups were ever attached, though, syncSubscribersOnSliceCreateOrUpdate
		// uses mcc+mnc as the serving PLMN key even when both are empty, so an all-zero PLMN can
		// already have subscriber records filed under that empty key -- changing away from it must
		// be rejected just like any other PLMN change, since the old records would be left in place.
		zeroPlmn := configmodels.SliceSiteInfoPlmn{}
		// This still infers exposure from the current device-group list rather than proof that no
		// record was ever written under the empty PLMN: handleNetworkSlicePost persists a slice
		// document before subscriber cleanup/sync for it is attempted, so a prior device-group
		// removal that updated this list but then failed cleanup could leave the list empty while
		// records remain. Closing that gap fully needs the document write and its subscriber
		// cleanup to be one transaction, which no write in this package currently is -- this is a
		// pre-existing, package-wide characteristic (see e.g. cleanupDeviceGroups), not something
		// specific to this check, and is tracked as a follow-up rather than fixed here.
		hadNoPriorSubscriberExposure := prevSlice.SiteInfo.Plmn == zeroPlmn && len(prevSlice.SiteDeviceGroup) == 0
		if requestSlice.SiteInfo.Plmn != prevSlice.SiteInfo.Plmn && !hadNoPriorSubscriberExposure {
			// The PLMN identifies the subscribers' serving network in every DB record keyed by
			// (imsi, PLMN). Changing it here would leave the old records in place and write new
			// ones under the new PLMN, duplicating every subscriber in the slice's device groups.
			err := fmt.Errorf("changing the PLMN (MCC/MNC) of an existing network slice %s is not allowed; delete and recreate the slice instead", sliceName)
			logger.ConfigLog.Errorln(err.Error())
			return http.StatusBadRequest, err
		}
		if statusCode, err := updateNS(requestSlice, *prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	}
	return http.StatusOK, nil
}

func parseAndValidateSliceRequest(c *gin.Context, sliceName string) (configmodels.Slice, int, error) {
	var request configmodels.Slice

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != jsonContentType {
		return request, http.StatusBadRequest, fmt.Errorf("unsupported content-type: %s", ct)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		return request, http.StatusBadRequest, fmt.Errorf("JSON bind error: %w", err)
	}

	for _, gnb := range request.SiteInfo.GNodeBs {
		if !isValidName(gnb.Name) {
			return request, http.StatusBadRequest, fmt.Errorf("invalid gNB name `%s` in Network Slice %s", gnb.Name, sliceName)
		}
		if !isValidGnbTac(gnb.Tac) {
			return request, http.StatusBadRequest, fmt.Errorf("invalid TAC %d for gNB %s in Network Slice %s", gnb.Tac, gnb.Name, sliceName)
		}
	}

	for _, ruleConfig := range request.ApplicationFilteringRules {
		if ruleConfig.TrafficClass == nil {
			logger.ConfigLog.Errorln("TrafficClass (QCI, ARP) required but not provided, network slice NOT configured in the network")
			return request, http.StatusBadRequest, fmt.Errorf("TrafficClass (QCI, ARP) required but not provided, network slice NOT configured in the network")
		}
		if err := validateRuleBitrates(ruleConfig, sliceName); err != nil {
			return request, http.StatusBadRequest, err
		}
	}

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)

	if statusCode, err := validateDeviceGroupsBelongToPlmn(request, sliceName); err != nil {
		return request, statusCode, err
	}

	return request, http.StatusOK, nil
}

// A slice's device groups must not carry subscribers from a different home network, since their
// records would then be filed under a serving PLMN they do not belong to.
func validateDeviceGroupsBelongToPlmn(request configmodels.Slice, sliceName string) (int, error) {
	mcc, mnc := request.SiteInfo.Plmn.Mcc, request.SiteInfo.Plmn.Mnc
	for _, dgName := range request.SiteDeviceGroup {
		devGroup, err := getDeviceGroupByName(dgName)
		if err != nil {
			// A lookup failure is not "no device group to validate": skipping validation on an
			// inconclusive read could let a mismatched IMSI through undetected. It is also not a
			// client error: the request itself may be perfectly valid.
			return http.StatusInternalServerError, fmt.Errorf("failed to look up device group %s: %w", dgName, err)
		}
		if devGroup == nil {
			continue
		}
		for _, imsi := range devGroup.Imsis {
			if !isValidImsiForPlmn(imsi, mcc, mnc) {
				return http.StatusBadRequest, fmt.Errorf("IMSI %s in device group %s does not belong to PLMN mcc=%s, mnc=%s of Network Slice %s", imsi, dgName, mcc, mnc, sliceName)
			}
		}
	}
	return http.StatusOK, nil
}

// A rate that isValidBitrate rejects is one that cannot be served as configured, so the slice is
// refused rather than accepted with a rate the operator never asked for.
func validateRuleBitrates(rule configmodels.SliceApplicationFilteringRules, sliceName string) error {
	rates := []struct {
		name  string
		value int32
	}{
		{"app-mbr-uplink", rule.AppMbrUplink},
		{"app-mbr-downlink", rule.AppMbrDownlink},
		{"app-gbr-uplink", rule.AppGbrUplink},
		{"app-gbr-downlink", rule.AppGbrDownlink},
	}
	for _, rate := range rates {
		if !isValidBitrate(rate.value, rule.BitrateUnit) {
			return fmt.Errorf("invalid %s %d %q for rule %s in Network Slice %s", rate.name, rate.value, rule.BitrateUnit, rule.RuleName, sliceName)
		}
	}
	return nil
}

func logSliceMetadata(slice configmodels.Slice) {
	logger.ConfigLog.Infof("network slice: sst: %s, sd: %s", slice.SliceId.Sst, slice.SliceId.Sd)
	logger.ConfigLog.Infof("number of device groups %v", len(slice.SiteDeviceGroup))
	for i, g := range slice.SiteDeviceGroup {
		logger.ConfigLog.Infof("device groups(%d) - %s", i+1, g)
	}

	site := slice.SiteInfo
	logger.ConfigLog.Infof("site name: %s", site.SiteName)
	logger.ConfigLog.Infof("site PLMN: mcc: %s, mnc: %s", site.Plmn.Mcc, site.Plmn.Mnc)
	for i, gnb := range site.GNodeBs {
		logger.ConfigLog.Infof("gNB (%d): name=%s, tac=%d", i+1, gnb.Name, gnb.Tac)
	}
	logger.ConfigLog.Infof("site UPF: %s", site.Upf)
}

func normalizeApplicationFilteringRules(slice *configmodels.Slice) {
	for i := range slice.ApplicationFilteringRules {
		rule := &slice.ApplicationFilteringRules[i]
		logger.ConfigLog.Infof("Rule [%d] Name: %s, Action: %s, Endpoint: %s", i, rule.RuleName, rule.Action, rule.Endpoint)

		ul := convertToBps(int64(rule.AppMbrUplink), rule.BitrateUnit)
		rule.AppMbrUplink = convertBitrateToInt32(ul)

		dl := convertToBps(int64(rule.AppMbrDownlink), rule.BitrateUnit)
		rule.AppMbrDownlink = convertBitrateToInt32(dl)

		// The guaranteed rates are expressed in the same unit and must be normalised the same way.
		// Left unconverted they would be stored raw, so a rule configured in Mbps would reach the
		// PCF a million times too small.
		gbrUl := convertToBps(int64(rule.AppGbrUplink), rule.BitrateUnit)
		rule.AppGbrUplink = convertBitrateToInt32(gbrUl)

		gbrDl := convertToBps(int64(rule.AppGbrDownlink), rule.BitrateUnit)
		rule.AppGbrDownlink = convertBitrateToInt32(gbrDl)

		// Every rate on the rule is bps from here on, so the unit has to say so. A GET returns the
		// stored rule, and returning the operator's original unit beside a normalised value both
		// contradicts the field description and multiplies the rates again if that document is
		// posted back.
		rule.BitrateUnit = bitrateUnitBps

		logger.ConfigLog.Infof("Normalized MBR Uplink: %d, Downlink: %d", rule.AppMbrUplink, rule.AppMbrDownlink)
		logger.ConfigLog.Infof("Normalized GBR Uplink: %d, Downlink: %d", rule.AppGbrUplink, rule.AppGbrDownlink)
		if rule.TrafficClass != nil {
			logger.ConfigLog.Infof("Traffic class: %v", rule.TrafficClass)
		}
	}
}

// labelStoredRatesAsBps makes a stored rule's unit describe the rates stored beside it.
//
// The rates in a rule are normalised to bps when it is written -- since 90de249 in November 2021,
// and by a fixed factor of a million before that -- so a stored value is bps whatever unit sits
// beside it, and every consumer of the stored rule, the policy served to the PCF included, reads
// it that way. Rules written before the unit was stored to match still carry the one the operator
// posted, so a GET would return a bps value labelled Kbps. That is not just a misleading label:
// posting the returned document back multiplies the rates again, a thousandfold for a rule
// configured in Kbps, and the operator has changed nothing.
//
// Rewriting the label on the way out rather than the rows in place keeps the read path honest
// without a migration, and a rule written since the ingest path started storing the unit is
// already bps, so this leaves it alone.
func labelStoredRatesAsBps(slice *configmodels.Slice) {
	for i := range slice.ApplicationFilteringRules {
		slice.ApplicationFilteringRules[i].BitrateUnit = bitrateUnitBps
	}
}

func convertBitrateToInt32(bitrate int64) int32 {
	if bitrate < 0 {
		logger.ConfigLog.Warnf("negative bitrate %d bps stored as 0", bitrate)
		return 0
	}
	if bitrate > math.MaxInt32 {
		logger.ConfigLog.Warnf("bitrate %d bps exceeds the largest rate that can be stored, capped at %d bps", bitrate, int64(math.MaxInt32))
		return math.MaxInt32
	}
	return int32(bitrate)
}

func createNS(slice configmodels.Slice) (int, error) {
	if statusCode, err := handleNetworkSlicePost(slice, configmodels.Slice{}); err != nil {
		logger.ConfigLog.Errorf("Error creating slice %s: %+v", slice.SliceName, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func updateNS(slice, prevSlice configmodels.Slice) (int, error) {
	if statusCode, err := handleNetworkSlicePost(slice, prevSlice); err != nil {
		logger.ConfigLog.Errorf("Error updating slice %s: %+v", slice.SliceName, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func handleNetworkSlicePost(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
	filter := bson.M{sliceNameKey: slice.SliceName}
	sliceDataBsonA := configmodels.ToBsonM(slice)
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(sliceDataColl, filter, sliceDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to post slice data for %s: %+v", slice.SliceName, err)
		return http.StatusInternalServerError, err
	}
	logger.DbLog.Debugf("succeeded to post slice data for %s", slice.SliceName)

	statusCode, err := syncSubscribersOnSliceCreateOrUpdate(slice, prevSlice)
	if err != nil {
		return statusCode, err
	}
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err = sendPebbleNotification("aetherproject.org/webconsole/networkslice/create")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return http.StatusOK, nil
}

func sendPebbleNotification(key string) error {
	cmd := execCommand("pebble", "notify", key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't execute a pebble notify: %w", err)
	}
	logger.ConfigLog.Infoln("custom Pebble notification sent")
	return nil
}

var syncSubscribersOnSliceDelete = func(slice *configmodels.Slice, prevSlice *configmodels.Slice) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	if slice == nil && prevSlice != nil {
		logger.WebUILog.Debugf("Deleted slice: %s", prevSlice.SliceName)
		return cleanupDeviceGroups(configmodels.Slice{}, *prevSlice)
	}
	return nil
}

var syncSubscribersOnSliceCreateOrUpdate = func(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
	rwLock.Lock()
	defer rwLock.Unlock()
	logger.WebUILog.Debugln("insert/update Slice:", slice)
	if slice.SliceId.Sst == "" {
		err := fmt.Errorf("missing SST in slice %s", slice.SliceName)
		logger.DbLog.Error(err)
		return http.StatusBadRequest, err
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.DbLog.Errorf("could not parse SST %s", slice.SliceId.Sst)
		return http.StatusBadRequest, err
	}
	snssai := models.NewSnssai(int32(sVal))
	snssai.SetSd(slice.SliceId.Sd)
	mcc := slice.SiteInfo.Plmn.Mcc
	mnc := slice.SiteInfo.Plmn.Mnc
	for _, dgName := range slice.SiteDeviceGroup {
		logger.ConfigLog.Debugf("dgName: %s", dgName)
		devGroupConfig, err := getDeviceGroupByName(dgName)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to look up device group %s: %w", dgName, err)
		}
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found: %s", dgName)
			continue
		}

		if len(devGroupConfig.IpDomainsExpanded) == 0 {
			logger.ConfigLog.Warnln("IPDomainExpanded is nil or empty for dgName:", dgName)
			continue
		}
		statusCode, err := processDeviceGroup(devGroupConfig, snssai, mcc, mnc)
		if err != nil {
			return statusCode, err
		}
	}
	if err := cleanupDeviceGroups(slice, prevSlice); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func processDeviceGroup(devGroupConfig *configmodels.DeviceGroups, snssai *models.Snssai, mcc, mnc string) (int, error) {
	dnnMap := make(map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) // Stores multiple DNNs & their QoS per IMSI
	for _, ipDomain := range devGroupConfig.IpDomainsExpanded {
		dnn := ipDomain.Dnn

		// Ensure UeDnnQos is not nil before appending
		if ipDomain.UeDnnQos != nil {
			// Append the QoS profile when present.
			dnnMap[dnn] = append(dnnMap[dnn], *ipDomain.UeDnnQos)
		} else {
			// Initialize an entry for this DNN if it doesn't exist so it is not omitted downstream.
			if _, exists := dnnMap[dnn]; !exists {
				dnnMap[dnn] = nil
			}
		}
	}
	var allQosProfiles []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos // Create a slice to hold all QoS profiles from all DNNs in the device group.
	// Iterate through the dnnMap to collect all QoS profiles into a single slice.
	for _, qosList := range dnnMap {
		allQosProfiles = append(allQosProfiles, qosList...)
	}
	// Calculate aggregate QoS once for the entire group
	aggregatedQoS := aggregateQoS(allQosProfiles)
	for i, imsi := range devGroupConfig.Imsis {
		// This is the authoritative check: parseAndValidateSliceRequest's pre-check ran before the
		// slice document was written and before rwLock (held by the caller) was acquired, so a
		// concurrent device-group update could have replaced this group's IMSIs in between.
		if !isValidImsiForPlmn(imsi, mcc, mnc) {
			return http.StatusBadRequest, fmt.Errorf("IMSI %s does not belong to PLMN mcc=%s, mnc=%s", imsi, mcc, mnc)
		}
		if subscriberAuthenticationDataGet("imsi-"+imsi) != nil {
			// Process each IP domain for this IMSI
			var gpsi string
			if devGroupConfig.Msisdns != nil && i < len(devGroupConfig.Msisdns) {
				gpsi = devGroupConfig.Msisdns[i]
			}
			// Call update functions once after processing all DNNs
			logger.ConfigLog.Infoln("Processing IMSI:", imsi, "with GPSI:", gpsi)
			err := updatePolicyAndProvisionedData(
				imsi,
				gpsi,
				snssai,
				dnnMap,
				mcc,
				mnc,
				aggregatedQoS,
			)
			if err != nil {
				logger.DbLog.Errorf("updatePolicyAndProvisionedData failed for IMSI %s: %+v", imsi, err)
				return http.StatusInternalServerError, err
			}
		}
	}
	return http.StatusOK, nil
}

func cleanupDeviceGroups(slice, prevSlice configmodels.Slice) error {
	dgnames := getDeletedDeviceGroupsList(slice, prevSlice)
	for _, dgName := range dgnames {
		devGroupConfig, err := getDeviceGroupByName(dgName)
		if err != nil {
			return fmt.Errorf("failed to look up device group %s during cleanup: %w", dgName, err)
		}
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found during cleanup: %s", dgName)
			continue
		}
		for _, imsi := range devGroupConfig.Imsis {
			mcc := prevSlice.SiteInfo.Plmn.Mcc
			mnc := prevSlice.SiteInfo.Plmn.Mnc
			if err := removeSubscriberEntriesRelatedToDeviceGroups(mcc, mnc, imsi); err != nil {
				logger.ConfigLog.Errorf("Failed to remove subscriber for IMSI %s: %+v", imsi, err)
				return err
			}
		}
	}
	return nil
}

func updatePolicyAndProvisionedData(imsi string, gpsi string, snssai *models.Snssai, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc string, mnc string, aggregatedQoS configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) error {
	err := updateAmPolicyData(imsi)
	if err != nil {
		return fmt.Errorf("updateAmPolicyData failed: %w", err)
	}
	err = updateSmPolicyData(snssai, dnnMap, imsi)
	if err != nil {
		return fmt.Errorf("updateSmPolicyData failed: %w", err)
	}
	err = updateAmProvisionedData(gpsi, snssai, aggregatedQoS, mcc, mnc, imsi)
	if err != nil {
		return fmt.Errorf("updateAmProvisionedData failed: %w", err)
	}
	err = updateSmProvisionedData(snssai, dnnMap, mcc, mnc, imsi)
	if err != nil {
		return fmt.Errorf("updateSmProvisionedData failed: %w", err)
	}
	err = updateSmfSelectionProvisionedData(snssai, mcc, mnc, dnnMap, imsi)
	if err != nil {
		return fmt.Errorf("updateSmfSelectionProvisionedData failed: %w", err)
	}
	return nil
}

func updateAmPolicyData(imsi string) error {
	var amPolicy models.AmPolicyData
	amPolicy.SubscCats = append(amPolicy.SubscCats, subscCatAether)
	amPolicyDatBsonA := configmodels.ToBsonM(amPolicy)
	amPolicyDatBsonA[ueIdKey] = "imsi-" + imsi
	filter := bson.M{ueIdKey: "imsi-" + imsi}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(amPolicyDataColl, filter, amPolicyDatBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update AM Policy Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update AM Policy Data for IMSI %s", imsi)
	return nil
}

func updateSmPolicyData(snssai *models.Snssai, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, imsi string) error {
	var smPolicyData models.SmPolicyData
	var smPolicySnssaiData models.SmPolicySnssaiData
	// Iterate over all DNNs in the map
	dnnData := &map[string]models.SmPolicyDnnData{}

	for dnn := range dnnMap { // Extract each DNN from the map
		(*dnnData)[dnn] = models.SmPolicyDnnData{
			Dnn: dnn,
		}
	}
	// smpolicydata
	smPolicySnssaiData.Snssai = *snssai
	smPolicySnssaiData.SmPolicyDnnData = dnnData
	smPolicyData.SmPolicySnssaiData = make(map[string]models.SmPolicySnssaiData)
	smPolicyData.SmPolicySnssaiData[SnssaiModelsToHex(*snssai)] = smPolicySnssaiData
	smPolicyDatBsonA := configmodels.ToBsonM(smPolicyData)
	smPolicyDatBsonA[ueIdKey] = "imsi-" + imsi
	filter := bson.M{ueIdKey: "imsi-" + imsi}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(smPolicyDataColl, filter, smPolicyDatBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update SM Policy Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update SM Policy Data for IMSI %s", imsi)
	return nil
}

func updateAmProvisionedData(gpsi string, snssai *models.Snssai, aggregatedQoS configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, imsi string) error {
	var gpsiSlice []string // Initialize a slice to hold the GPSI.
	if gpsi != "" {        // Only add if gpsi is not empty
		gpsiSlice = []string{gpsi}
	}
	amData := models.AccessAndMobilitySubscriptionData{
		Gpsis: gpsiSlice,
		Nssai: *models.NewNullableNssai(&models.Nssai{
			DefaultSingleNssais: []models.Snssai{*snssai},
			SingleNssais:        []models.Snssai{*snssai},
		}),
		SubscribedUeAmbr: models.NewAmbr(ConvertToString(uint64(aggregatedQoS.DnnMbrUplink)), ConvertToString(uint64(aggregatedQoS.DnnMbrDownlink))),
	}
	amDataBsonA := configmodels.ToBsonM(amData)
	amDataBsonA[ueIdKey] = "imsi-" + imsi
	amDataBsonA[servingPlmnIdKey] = mcc + mnc
	filter := bson.M{
		ueIdKey: "imsi-" + imsi,
		"$or": []bson.M{
			{servingPlmnIdKey: mcc + mnc},
			{servingPlmnIdKey: bson.M{"$exists": false}},
		},
	}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(amDataColl, filter, amDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update AM provisioned Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update AM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmProvisionedData(snssai *models.Snssai, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, imsi string) error {
	filter := bson.M{
		ueIdKey:          "imsi-" + imsi,
		servingPlmnIdKey: mcc + mnc,
	}

	smDataBsonA, err := buildSmProvisionedDataDocument(snssai, dnnMap, mcc, mnc, imsi)
	if err != nil {
		return err
	}

	logger.DbLog.Infof("Data to be sent to database - SmProvisionedData: %+v", smDataBsonA)
	_, errPut := dbadapter.CommonDBClient.RestfulAPIPutOne(smDataColl, filter, smDataBsonA)
	if errPut != nil {
		logger.DbLog.Errorf("failed to update SM provisioned Data for IMSI %s: %+v", imsi, errPut)
		return errPut
	}
	logger.DbLog.Debugf("updated SM provisioned Data for IMSI %s", imsi)
	return nil
}

func buildSmProvisionedDataDocument(snssai *models.Snssai, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, imsi string) (map[string]interface{}, error) {
	dnnConfigurations := make(map[string]interface{}, len(dnnMap))

	for dnn, ueDnnQosList := range dnnMap {
		aggregatedQoS := aggregateQoS(ueDnnQosList)
		if aggregatedQoS.TrafficClass == nil {
			logger.DbLog.Errorf("TrafficClass is nil for DNN %s, IMSI %s", dnn, imsi)
			return nil, fmt.Errorf("traffic class missing for DNN %s", dnn)
		}

		dnnConfigurations[dnn] = map[string]interface{}{
			"pduSessionTypes": map[string]interface{}{
				"defaultSessionType":  models.PDUSESSIONTYPE_IPV4,
				"allowedSessionTypes": []models.PduSessionType{models.PDUSESSIONTYPE_IPV4},
			},
			"sscModes": map[string]interface{}{
				"defaultSscMode":  models.SSCMODE_SSC_MODE_1,
				"allowedSscModes": []models.SscMode{models.SSCMODE_SSC_MODE_2, models.SSCMODE_SSC_MODE_3},
			},
			"sessionAmbr": map[string]interface{}{
				downlinkKey: ConvertToString(uint64(aggregatedQoS.DnnMbrDownlink)),
				uplinkKey:   ConvertToString(uint64(aggregatedQoS.DnnMbrUplink)),
			},
			"5gQosProfile": map[string]interface{}{
				"5qi": aggregatedQoS.TrafficClass.Qci,
				"arp": map[string]interface{}{
					priorityLevelKey: int32(8),
					"preemptCap":     models.PREEMPTIONCAPABILITY_NOT_PREEMPT,
					"preemptVuln":    models.PREEMPTIONVULNERABILITY_NOT_PREEMPTABLE,
				},
				priorityLevelKey: int32(8),
			},
		}
	}

	singleNssai := map[string]interface{}{
		"sst": snssai.Sst,
	}
	if snssai.Sd != nil {
		singleNssai["sd"] = *snssai.Sd
	}

	return map[string]interface{}{
		ueIdKey:             "imsi-" + imsi,
		servingPlmnIdKey:    mcc + mnc,
		"singlenssai":       singleNssai,
		"dnnconfigurations": dnnConfigurations,
	}, nil
}

func updateSmfSelectionProvisionedData(snssai *models.Snssai, mcc, mnc string, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, imsi string) error {
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: &map[string]models.SnssaiInfo{},
	}

	snssaiInfo := models.SnssaiInfo{
		DnnInfos: []models.DnnInfo{},
	}
	// Collect DNN keys
	dnns := make([]string, 0, len(dnnMap))
	for dnn := range dnnMap {
		dnns = append(dnns, dnn)
	}

	// Sort for deterministic ordering
	sort.Strings(dnns)

	// Append in sorted order
	for _, dnn := range dnns {
		snssaiInfo.DnnInfos = append(snssaiInfo.DnnInfos, models.DnnInfo{
			Dnn: models.AccessAndMobilitySubscriptionDataSubscribedDnnListInner{
				String: openapi.PtrString(dnn),
			},
		})
	}
	(*smfSelData.SubscribedSnssaiInfos)[SnssaiModelsToHex(*snssai)] = snssaiInfo
	smfSelecDataBsonA := configmodels.ToBsonM(smfSelData)
	smfSelecDataBsonA[ueIdKey] = "imsi-" + imsi
	smfSelecDataBsonA[servingPlmnIdKey] = mcc + mnc

	// Define the filter for the database operation
	filter := bson.M{
		ueIdKey:          "imsi-" + imsi,
		servingPlmnIdKey: mcc + mnc,
	}

	// Log the data to be sent to the database
	logger.DbLog.Infof("Data to be sent to database - smf selection: %+v", smfSelecDataBsonA)

	// Perform the database post operation
	_, errPost := dbadapter.CommonDBClient.RestfulAPIPost(smfSelDataColl, filter, smfSelecDataBsonA)
	if errPost != nil {
		logger.DbLog.Errorf("failed to update SMF selection provisioned data for IMSI %s: %+v", imsi, errPost)
		return errPost
	}
	logger.DbLog.Debugf("updated SMF selection provisioned data for IMSI %s", imsi)
	return nil
}

func SnssaiModelsToHex(snssai models.Snssai) string {
	sst := fmt.Sprintf("%02x", snssai.Sst)
	return sst + snssai.GetSd()
}

// ConvertToString renders a rate held in bps for the policy served to the PCF.
//
// The unit has to be one the consumers can read and the numeral has to be one they can hold.
// omec-project/smf turns these strings into the QoS flow description the UE is signalled, and
// GetBitRate there switches on the unit with no case for bps -- the default arm is Mbps, so
// "1500 bps" would tell the UE 1500 Mbps, a million times the rate configured. It then parses the
// numeral into a uint16, which "65536 Mbps" reaches the UE as 0 and "2147000 Kbps" as 49848. The
// Session-AMBR converter in omec-project/nas has the same uint16 limit on the numeral.
//
// So the unit is the largest one that describes the rate exactly and whose numeral fits, and
// failing that the smallest one whose numeral fits, which is the one that truncates least. An
// exact unit is refused when its numeral does not fit: 65536000 bps is a whole number of kbps and
// is still served as "65 Mbps".
//
// Two bands have no unit to fall back to and are rendered in bps as they always were: below a
// kbps, and above what a Gbps numeral holds. Only the device-group rates reach the second, their
// int64 field holding either the math.MaxInt64 the ingest path clamps a negative rate to or a
// product of its own conversion that wrapped. Both are defects of their own.
//
// The exactness is what the old integer division lost -- 2147000000 bps was served as "2 Gbps",
// 147 Mbps below the rate configured, where "2147 Mbps" describes it exactly. For a maximum rate
// that is a ceiling under the one asked for; for a guaranteed rate it is a floor the network never
// commits to.
func ConvertToString(val uint64) string {
	switch {
	case isReadableAndExact(val, GBPS):
		return strconv.FormatUint(val/GBPS, 10) + " Gbps"
	case isReadableAndExact(val, MBPS):
		return strconv.FormatUint(val/MBPS, 10) + " Mbps"
	case isReadableAndExact(val, KBPS):
		return strconv.FormatUint(val/KBPS, 10) + " Kbps"
	case isReadable(val, KBPS):
		return strconv.FormatUint(val/KBPS, 10) + " Kbps"
	case isReadable(val, MBPS):
		return strconv.FormatUint(val/MBPS, 10) + " Mbps"
	case isReadable(val, GBPS):
		return strconv.FormatUint(val/GBPS, 10) + " Gbps"
	default:
		return strconv.FormatUint(val, 10) + " bps"
	}
}

// maxReadableBitRateNumeral is the largest numeral the consumers of these strings can hold: both
// smf's GetBitRate and nas's Session-AMBR converter read it into a uint16.
const maxReadableBitRateNumeral = 65535

// isReadable reports whether val is at least one of the given unit and whose numeral in that unit
// the consumers can hold.
func isReadable(val, unit uint64) bool {
	return val >= unit && val/unit <= maxReadableBitRateNumeral
}

// isReadableAndExact adds that the unit describes the rate with nothing left over.
func isReadableAndExact(val, unit uint64) bool {
	return isReadable(val, unit) && val%unit == 0
}

// getSlices returns every slice, and an error if the read or an unmarshal failed -- callers must
// not treat a failure as "no slices exist", since that would make a PLMN/association check that
// depends on the full slice collection silently pass instead of failing closed.
func getSlices() ([]*configmodels.Slice, error) {
	rawSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, nil)
	if err != nil {
		logger.DbLog.Warnln(err)
		return nil, err
	}
	var slices []*configmodels.Slice
	for _, rawSlice := range rawSlices {
		var sliceData configmodels.Slice
		if err := json.Unmarshal(configmodels.MapToByte(rawSlice), &sliceData); err != nil {
			logger.DbLog.Errorf("could not unmarshall slice %+v", rawSlice)
			return nil, err
		}
		slices = append(slices, &sliceData)
	}
	return slices, nil
}

// getSliceByName returns (nil, nil) when no slice matches name, and (nil, err) when the lookup or
// its unmarshal failed -- the two must stay distinguishable so a transient read failure is never
// mistaken for "slice does not exist yet" by a caller that would otherwise create/overwrite it.
func getSliceByName(name string) (*configmodels.Slice, error) {
	filter := bson.M{sliceNameKey: name}
	sliceDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(sliceDataColl, filter)
	if err != nil {
		logger.DbLog.Warnln(err)
		return nil, err
	}
	if sliceDataInterface == nil {
		return nil, nil
	}
	var sliceData configmodels.Slice
	if err := json.Unmarshal(configmodels.MapToByte(sliceDataInterface), &sliceData); err != nil {
		logger.DbLog.Errorf("could not unmarshall slice %+v", sliceDataInterface)
		return nil, err
	}
	return &sliceData, nil
}

func handleNetworkSliceDelete(sliceName string) error {
	prevSlice, err := getSliceByName(sliceName)
	if err != nil {
		// The previous slice is required for subscriber cleanup below; deleting without it would
		// leave AM/SM/SMF-selection records orphaned under the slice's old PLMN.
		return fmt.Errorf("failed to look up slice %s before delete: %w", sliceName, err)
	}
	filter := bson.M{sliceNameKey: sliceName}
	err = dbadapter.CommonDBClient.RestfulAPIDeleteOne(sliceDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to delete slice data for %+v: %+v", sliceName, err)
		return err
	}
	// slice is nil as it is deleted
	if err = syncSubscribersOnSliceDelete(nil, prevSlice); err != nil {
		logger.WebUILog.Errorf("failed to cleanup subscriber entries related to device groups %+v: %+v", sliceName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to delete slice data for %s", sliceName)
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err = sendPebbleNotification("aetherproject.org/webconsole/networkslice/delete")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return nil
}

func getDeletedDeviceGroupsList(slice, prevSlice configmodels.Slice) []string {
	if len(prevSlice.SiteDeviceGroup) == 0 {
		return nil
	}
	if len(slice.SiteDeviceGroup) == 0 {
		return slices.Clone(prevSlice.SiteDeviceGroup)
	}

	var deleted []string
	for _, pdgName := range prevSlice.SiteDeviceGroup {
		if !slices.Contains(slice.SiteDeviceGroup, pdgName) {
			deleted = append(deleted, pdgName)
		}
	}
	return deleted
}

// addBitrateBps sums two rates in bps without wrapping, and saturates at the largest rate
// ConvertToString can express, since the aggregate is served the same way a single one is.
//
// The aggregate is the sum of every IP domain in a device group, so it can exceed the bound each
// rate is validated against on its own. It could also reach it from below: a group written before
// the rates were bounded holds the math.MaxInt64 the old ingest path clamped a negative rate to,
// and two of those summed plainly give -2, which is served as a rate of 18446744073709551614 bps.
// Each operand is brought into range before it is added, so the sum cannot wrap.
func addBitrateBps(a, b int64) int64 {
	return ClampDeviceGroupBitrateBps(ClampDeviceGroupBitrateBps(a) + ClampDeviceGroupBitrateBps(b))
}

// ClampDeviceGroupBitrateBps bounds a device group rate to what ConvertToString can render, so a
// legacy value read straight from storage -- not yet corrected by validateUeDnnQosBitrates because
// it predates that check -- cannot be served as an unreadable or wrapped rate.
func ClampDeviceGroupBitrateBps(val int64) int64 {
	if val < 0 {
		return 0
	}
	if val > maxDeviceGroupBitrateBps {
		return maxDeviceGroupBitrateBps
	}
	return val
}

func aggregateQoS(qosList []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) configmodels.DeviceGroupsIpDomainExpandedUeDnnQos {
	var aggregated configmodels.DeviceGroupsIpDomainExpandedUeDnnQos

	if len(qosList) == 0 {
		logger.ConfigLog.Debugln("aggregateQoS called with empty qosList")
		return aggregated
	}

	// Track bitrate unit consistency
	var firstNonEmptyUnit string
	for _, qos := range qosList {
		if qos.BitrateUnit != "" {
			firstNonEmptyUnit = qos.BitrateUnit
			break
		}
	}

	unitConsistent := true

	for _, qos := range qosList {
		aggregated.DnnMbrUplink = addBitrateBps(aggregated.DnnMbrUplink, qos.DnnMbrUplink)
		aggregated.DnnMbrDownlink = addBitrateBps(aggregated.DnnMbrDownlink, qos.DnnMbrDownlink)

		// Warn if units are inconsistent (ignoring empty units)
		if qos.BitrateUnit != "" && firstNonEmptyUnit != "" && qos.BitrateUnit != firstNonEmptyUnit {
			unitConsistent = false
		}

		// Use the first valid bitrate unit
		if aggregated.BitrateUnit == "" && qos.BitrateUnit != "" {
			aggregated.BitrateUnit = qos.BitrateUnit
		}

		// Use the first non-nil traffic class (prefer higher priority)
		if qos.TrafficClass != nil {
			if aggregated.TrafficClass == nil {
				aggregated.TrafficClass = qos.TrafficClass
			} else if qos.TrafficClass.Qci < aggregated.TrafficClass.Qci {
				// Lower QCI value = higher priority, use higher priority class
				aggregated.TrafficClass = qos.TrafficClass
				logger.ConfigLog.Infof("using higher priority QoS class (QCI %d)", qos.TrafficClass.Qci)
			}
		}
	}

	if !unitConsistent {
		logger.ConfigLog.Warnf("inconsistent bitrate units detected when aggregating QoS (using %s)", aggregated.BitrateUnit)
	}

	return aggregated
}
