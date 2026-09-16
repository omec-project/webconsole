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
	requestSlice, err := parseAndValidateSliceRequest(c, sliceName)
	if err != nil {
		return http.StatusBadRequest, err
	}

	logSliceMetadata(requestSlice)
	normalizeApplicationFilteringRules(&requestSlice)
	requestSlice.SliceName = sliceName
	prevSlice := getSliceByName(sliceName)

	if prevSlice == nil {
		logger.ConfigLog.Infof("Adding new slice [%s]", sliceName)
		if statusCode, err := createNS(requestSlice); err != nil {
			logger.ConfigLog.Errorf("Error creating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	} else {
		if statusCode, err := updateNS(requestSlice, *prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	}
	return http.StatusOK, nil
}

func parseAndValidateSliceRequest(c *gin.Context, sliceName string) (configmodels.Slice, error) {
	var request configmodels.Slice

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != jsonContentType {
		return request, fmt.Errorf("unsupported content-type: %s", ct)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		return request, fmt.Errorf("JSON bind error: %w", err)
	}

	for _, gnb := range request.SiteInfo.GNodeBs {
		if !isValidName(gnb.Name) {
			return request, fmt.Errorf("invalid gNB name `%s` in Network Slice %s", gnb.Name, sliceName)
		}
		if !isValidGnbTac(gnb.Tac) {
			return request, fmt.Errorf("invalid TAC %d for gNB %s in Network Slice %s", gnb.Tac, gnb.Name, sliceName)
		}
	}

	for _, ruleConfig := range request.ApplicationFilteringRules {
		if ruleConfig.TrafficClass == nil {
			logger.ConfigLog.Errorln("TrafficClass (QCI, ARP) required but not provided, network slice NOT configured in the network")
			return request, fmt.Errorf("TrafficClass (QCI, ARP) required but not provided, network slice NOT configured in the network")
		}
		if err := validateRuleBitrates(ruleConfig, sliceName); err != nil {
			return request, err
		}
	}

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)

	return request, nil
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
		devGroupConfig := getDeviceGroupByName(dgName)
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found: %s", dgName)
			continue
		}

		if len(devGroupConfig.IpDomainsExpanded) == 0 {
			logger.ConfigLog.Warnln("IPDomainExpanded is nil or empty for dgName:", dgName)
			continue
		}
		_, err := processDeviceGroup(devGroupConfig, snssai, mcc, mnc)
		if err != nil {
			return http.StatusInternalServerError, err
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
		devGroupConfig := getDeviceGroupByName(dgName)
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
// It names the largest unit that describes the rate exactly -- 2147000000 bps is "2147 Mbps" and
// not the "2 Gbps" that integer division used to serve, 147 Mbps below the rate configured -- and
// falls back to a truncated Kbps where no unit describes it exactly.
//
// That fallback is not what this would do if the choice were free: the exact answer for 1500 bps
// is "1500 bps" and the 3GPP bit rate format allows it. It is what the consumers can read.
// omec-project/smf turns these strings into the QoS flow description the UE is signalled, and
// GetBitRate there switches on the unit with no case for bps -- the default arm is Mbps, so
// "1500 bps" tells the UE 1500 Mbps, a million times the rate configured. The Session-AMBR
// converter in omec-project/nas does know the unit, but maps it to "unit not used" and parses the
// numeric as a uint16, so a bps rate of 65536 or more is encoded as zero.
//
// A rate below a kbps has no smaller unit to fall back to and is still rendered in bps, as it
// always was.
func ConvertToString(val uint64) string {
	switch {
	case val != 0 && val%1000000000 == 0:
		return strconv.FormatUint(val/1000000000, 10) + " Gbps"
	case val != 0 && val%1000000 == 0:
		return strconv.FormatUint(val/1000000, 10) + " Mbps"
	case val >= 1000:
		return strconv.FormatUint(val/1000, 10) + " Kbps"
	default:
		return strconv.FormatUint(val, 10) + " bps"
	}
}

func getSlices() []*configmodels.Slice {
	rawSlices, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, nil)
	if errGetMany != nil {
		logger.DbLog.Warnln(errGetMany)
	}
	var slices []*configmodels.Slice
	for _, rawSlice := range rawSlices {
		var sliceData configmodels.Slice
		err := json.Unmarshal(configmodels.MapToByte(rawSlice), &sliceData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshall slice %+v", rawSlice)
		}
		slices = append(slices, &sliceData)
	}
	return slices
}

func getSliceByName(name string) *configmodels.Slice {
	filter := bson.M{sliceNameKey: name}
	sliceDataInterface, errGetOne := dbadapter.CommonDBClient.RestfulAPIGetOne(sliceDataColl, filter)
	if errGetOne != nil {
		logger.DbLog.Warnln(errGetOne)
		return nil
	}
	var sliceData configmodels.Slice
	err := json.Unmarshal(configmodels.MapToByte(sliceDataInterface), &sliceData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall slice %+v", sliceDataInterface)
		return nil
	}
	return &sliceData
}

func handleNetworkSliceDelete(sliceName string) error {
	prevSlice := getSliceByName(sliceName)
	filter := bson.M{sliceNameKey: sliceName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(sliceDataColl, filter)
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
		aggregated.DnnMbrUplink += qos.DnnMbrUplink
		aggregated.DnnMbrDownlink += qos.DnnMbrDownlink

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
