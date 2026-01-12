// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
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
	if ct != "application/json" {
		return request, fmt.Errorf("unsupported content-type: %s", ct)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		return request, fmt.Errorf("JSON bind error: %+v", err)
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
	}

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)

	return request, nil
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

		logger.ConfigLog.Infof("Normalized MBR Uplink: %v, Downlink: %v", rule.AppMbrUplink, rule.AppMbrDownlink)
		if rule.TrafficClass != nil {
			logger.ConfigLog.Infof("Traffic class: %v", rule.TrafficClass)
		}
	}
}

func convertBitrateToInt32(bitrate int64) int32 {
	if bitrate < 0 || bitrate > math.MaxInt32 {
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
	filter := bson.M{"slice-name": slice.SliceName}
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
		return fmt.Errorf("couldn't execute a pebble notify: %+v", err)
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
	snssai := &models.Snssai{
		Sd:  slice.SliceId.Sd,
		Sst: int32(sVal),
	}
	mcc := slice.SiteInfo.Plmn.Mcc
	mnc := slice.SiteInfo.Plmn.Mnc
	for _, dgName := range slice.SiteDeviceGroup {
		logger.ConfigLog.Debugf("dgName: %s", dgName)
		devGroupConfig := getDeviceGroupByName(dgName)
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found: %s", dgName)
			continue
		}

		if len(devGroupConfig.IpDomainExpanded) == 0 {
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
	for _, ipDomain := range devGroupConfig.IpDomainExpanded {
		dnn := ipDomain.Dnn

		// Ensure UeDnnQos is not nil before appending
		if ipDomain.UeDnnQos != nil {
			dnnMap[dnn] = append(dnnMap[dnn], *ipDomain.UeDnnQos) // Directly append the UeDnnQos
		}
	}
	var allQosProfiles []configmodels.DeviceGroupsIpDomainExpandedUeDnnQos // Create a slice to hold all QoS profiles from all DNNs in the device group.
	// Iterate through the dnnMap to collect all QoS profiles into a single slice.
	for _, qosList := range dnnMap {
		allQosProfiles = append(allQosProfiles, qosList...)
	}
	// Calculate aggragate QoS once for the entire group
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
	amPolicy.SubscCats = append(amPolicy.SubscCats, "aether")
	amPolicyDatBsonA := configmodels.ToBsonM(amPolicy)
	amPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
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
	dnnData := make(map[string]models.SmPolicyDnnData)

	for dnn := range dnnMap { // Extract each DNN from the map
		dnnData[dnn] = models.SmPolicyDnnData{
			Dnn: dnn,
		}
	}
	// smpolicydata
	smPolicySnssaiData.Snssai = snssai
	smPolicySnssaiData.SmPolicyDnnData = dnnData
	smPolicyData.SmPolicySnssaiData = make(map[string]models.SmPolicySnssaiData)
	smPolicyData.SmPolicySnssaiData[SnssaiModelsToHex(*snssai)] = smPolicySnssaiData
	smPolicyDatBsonA := configmodels.ToBsonM(smPolicyData)
	smPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
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
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{*snssai},
			SingleNssais:        []models.Snssai{*snssai},
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: ConvertToString(uint64(aggregatedQoS.DnnMbrDownlink)),
			Uplink:   ConvertToString(uint64(aggregatedQoS.DnnMbrUplink)),
		},
	}
	amDataBsonA := configmodels.ToBsonM(amData)
	amDataBsonA["ueId"] = "imsi-" + imsi
	amDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{
		"ueId": "imsi-" + imsi,
		"$or": []bson.M{
			{"servingPlmnId": mcc + mnc},
			{"servingPlmnId": bson.M{"$exists": false}},
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
	// Define the filter to find the existing record for this UE
	filter := bson.M{
		"ueId":          "imsi-" + imsi,
		"servingPlmnId": mcc + mnc,
	}

	// Fetch the existing record from the database
	existingRecord, err := dbadapter.CommonDBClient.RestfulAPIGetOne(smDataColl, filter)
	if err != nil && err.Error() != "mongo: no documents in result" {
		logger.DbLog.Warnf("Failed to fetch existing record for ueId: %s, error: %v", imsi, err)
		return err
	}
	var smData models.SessionManagementSubscriptionData
	if existingRecord == nil {
		// No existing record, create a new one
		smData = models.SessionManagementSubscriptionData{
			SingleNssai:       snssai,
			DnnConfigurations: make(map[string]models.DnnConfiguration),
		}
	} else {
		bsonBytes, errMarshal := bson.Marshal(existingRecord) // Use a different name for error
		if errMarshal != nil {
			logger.DbLog.Errorf("Failed to marshal existing record: %v", errMarshal)
			return errMarshal
		}

		// Unmarshal BSON into struct
		errUnmarshal := bson.Unmarshal(bsonBytes, &smData) // Use a different name for error
		if errUnmarshal != nil {
			logger.DbLog.Errorf("Failed to unmarshal existing record: %v", errUnmarshal)
			return errUnmarshal
		}
	}
	for dnn, ueDnnQosList := range dnnMap {
		aggregatedQoS := aggregateQoS(ueDnnQosList)

		smData.DnnConfigurations[dnn] = models.DnnConfiguration{
			PduSessionTypes: &models.PduSessionTypes{
				DefaultSessionType:  models.PduSessionType_IPV4,
				AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
			},
			SscModes: &models.SscModes{
				DefaultSscMode: models.SscMode__1,
				AllowedSscModes: []models.SscMode{
					"SSC_MODE_2",
					"SSC_MODE_3",
				},
			},
			SessionAmbr: &models.Ambr{
				Downlink: ConvertToString(uint64(aggregatedQoS.DnnMbrDownlink)),
				Uplink:   ConvertToString(uint64(aggregatedQoS.DnnMbrUplink)),
			},
			Var5gQosProfile: &models.SubscribedDefaultQos{
				Var5qi: aggregatedQoS.TrafficClass.Qci,
				Arp: &models.Arp{
					PriorityLevel: 8,
				},
				PriorityLevel: 8,
			},
		}
	}

	bsonBytes, err := bson.Marshal(smData)
	if err != nil {
		logger.DbLog.Errorf("Failed to marshal smData: %v", err)
		return err
	}

	var smDataBsonA map[string]interface{}
	err = bson.Unmarshal(bsonBytes, &smDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("Failed to unmarshal smData BSON: %v", err)
		return err
	}

	// Add required fields
	smDataBsonA["ueId"] = "imsi-" + imsi
	smDataBsonA["servingPlmnId"] = mcc + mnc

	// Update the database
	logger.DbLog.Infof("Data to be sent to database - SmProvisionedData: %+v", smDataBsonA)
	_, errPost := dbadapter.CommonDBClient.RestfulAPIPost(smDataColl, filter, smDataBsonA)
	if errPost != nil {
		logger.DbLog.Errorf("failed to update SM provisioned Data for IMSI %s: %+v", imsi, err)
		return errPost
	}
	logger.DbLog.Debugf("updated SM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmfSelectionProvisionedData(snssai *models.Snssai, mcc, mnc string, dnnMap map[string][]configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, imsi string) error {
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{},
	}

	snssaiInfo := models.SnssaiInfo{
		DnnInfos: []models.DnnInfo{},
	}

	for dnn := range dnnMap {
		// Append each DNN's info to DnnInfos
		snssaiInfo.DnnInfos = append(snssaiInfo.DnnInfos, models.DnnInfo{
			Dnn: dnn,
		})
	}
	smfSelData.SubscribedSnssaiInfos[SnssaiModelsToHex(*snssai)] = snssaiInfo
	smfSelecDataBsonA := configmodels.ToBsonM(smfSelData)
	smfSelecDataBsonA["ueId"] = "imsi-" + imsi
	smfSelecDataBsonA["servingPlmnId"] = mcc + mnc

	// Define the filter for the database operation
	filter := bson.M{
		"ueId":          "imsi-" + imsi,
		"servingPlmnId": mcc + mnc,
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
	return sst + snssai.Sd
}

func ConvertToString(val uint64) string {
	var mbVal, gbVal, kbVal uint64
	kbVal = val / 1000
	mbVal = val / 1000000
	gbVal = val / 1000000000
	var retStr string
	if gbVal != 0 {
		retStr = strconv.FormatUint(gbVal, 10) + " Gbps"
	} else if mbVal != 0 {
		retStr = strconv.FormatUint(mbVal, 10) + " Mbps"
	} else if kbVal != 0 {
		retStr = strconv.FormatUint(kbVal, 10) + " Kbps"
	} else {
		retStr = strconv.FormatUint(val, 10) + " bps"
	}

	return retStr
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
	filter := bson.M{"slice-name": name}
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
	filter := bson.M{"slice-name": sliceName}
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
	for _, qos := range qosList {
		aggregated.DnnMbrUplink += qos.DnnMbrUplink
		aggregated.DnnMbrDownlink += qos.DnnMbrDownlink
		aggregated.BitrateUnit = qos.BitrateUnit
		if qos.TrafficClass != nil {
			aggregated.TrafficClass = qos.TrafficClass
		}
	}
	return aggregated
}
