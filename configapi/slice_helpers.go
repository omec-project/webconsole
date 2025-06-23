package configapi

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	execCommand = exec.Command
)

func handleNetworkSlicePost(slice *configmodels.Slice, prevSlice *configmodels.Slice) error {
	filter := bson.M{"slice-name": slice.SliceName}
	sliceDataBsonA := configmodels.ToBsonM(slice)
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(sliceDataColl, filter, sliceDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to post slice data for %v: %v", slice.SliceName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to post slice data for %v", slice.SliceName)

	err = syncSliceDeviceGroupSubscribers(slice, prevSlice)
	if err != nil {
		return err
	}
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err := sendPebbleNotification("aetherproject.org/webconsole/networkslice/create")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return nil
}

func sendPebbleNotification(key string) error {
	cmd := execCommand("pebble", "notify", key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't execute a pebble notify: %w", err)
	}
	logger.ConfigLog.Infoln("custom Pebble notification sent")
	return nil
}

var syncSliceDeviceGroupSubscribers = func(slice *configmodels.Slice, prevSlice *configmodels.Slice) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	logger.WebUILog.Debugln("insert/update Network Slice")
	mongoClient := dbadapter.CommonDBClient.(*mongoapi.MongoClient)
	sessionRunner := dbadapter.RealSessionRunner(mongoClient.Client)

	if slice == nil && prevSlice != nil {
		logger.WebUILog.Debugln("deleted Slice:", prevSlice)
	}
	if slice != nil {
		logger.WebUILog.Debugln("insert/update Slice:", slice)
		sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
		if err != nil {
			logger.DbLog.Errorf("could not parse SST %v", slice.SliceId.Sst)
			return err
		}
		snssai := &models.Snssai{
			Sd:  slice.SliceId.Sd,
			Sst: int32(sVal),
		}
		for _, dgName := range slice.SiteDeviceGroup {
			logger.ConfigLog.Infoln("dgName:", dgName)
			devGroupConfig := getDeviceGroupByName(dgName)
			if devGroupConfig != nil {
				for _, imsi := range devGroupConfig.Imsis {
					dnn := devGroupConfig.IpDomainExpanded.Dnn
					mcc := slice.SiteInfo.Plmn.Mcc
					mnc := slice.SiteInfo.Plmn.Mnc
					err = updatePolicyAndProvisionedData(
						imsi,
						mcc,
						mnc,
						snssai,
						dnn,
						devGroupConfig.IpDomainExpanded.UeDnnQos,
					)
					if err != nil {
						logger.DbLog.Errorf("updatePolicyAndProvisionedData failed for IMSI %s: %v", imsi, err)
						return err
					}
				}
			}
		}
	}

	dgnames := getDeletedDeviceGroupsList(slice, prevSlice)
	for _, dgname := range dgnames {
		devGroupConfig := getDeviceGroupByName(dgname)
		if devGroupConfig != nil {
			for _, imsi := range devGroupConfig.Imsis {
				err := removeSubscriberEntriesRelatedToDeviceGroups(prevSlice.SiteInfo.Plmn.Mcc, prevSlice.SiteInfo.Plmn.Mnc, imsi, sessionRunner)
				if err != nil {
					logger.ConfigLog.Errorln(err)
					return err
				}
			}
		}
	}
	return nil
}

func updatePolicyAndProvisionedData(imsi string, mcc string, mnc string, snssai *models.Snssai, dnn string, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) error {
	err := updateAmPolicyData(imsi)
	if err != nil {
		return fmt.Errorf("updateAmPolicyData failed: %w", err)
	}
	err = updateSmPolicyData(snssai, dnn, imsi)
	if err != nil {
		return fmt.Errorf("updateSmPolicyData failed: %w", err)
	}
	err = updateAmProvisionedData(snssai, qos, mcc, mnc, imsi)
	if err != nil {
		return fmt.Errorf("updateAmProvisionedData failed: %w", err)
	}
	err = updateSmProvisionedData(snssai, qos, mcc, mnc, dnn, imsi)
	if err != nil {
		return fmt.Errorf("updateSmProvisionedData failed: %w", err)
	}
	err = updateSmfSelectionProvisionedData(snssai, mcc, mnc, dnn, imsi)
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
		logger.DbLog.Errorf("failed to update AM Policy Data for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update AM Policy Data for IMSI %s", imsi)
	return nil
}

func updateSmPolicyData(snssai *models.Snssai, dnn string, imsi string) error {
	var smPolicyData models.SmPolicyData
	var smPolicySnssaiData models.SmPolicySnssaiData
	dnnData := map[string]models.SmPolicyDnnData{
		dnn: {
			Dnn: dnn,
		},
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
		logger.DbLog.Errorf("failed to update SM Policy Data for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update SM Policy Data for IMSI %s", imsi)
	return nil
}

func updateAmProvisionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, imsi string) error {
	amData := models.AccessAndMobilitySubscriptionData{
		Gpsis: []string{
			"msisdn-0900000000",
		},
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{*snssai},
			SingleNssais:        []models.Snssai{*snssai},
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: convertToString(uint64(qos.DnnMbrDownlink)),
			Uplink:   convertToString(uint64(qos.DnnMbrUplink)),
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
		logger.DbLog.Errorf("failed to update AM provisioned Data for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to update AM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmProvisionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string) error {
	smData := models.SessionManagementSubscriptionData{
		SingleNssai: snssai,
		DnnConfigurations: map[string]models.DnnConfiguration{
			dnn: {
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
					Downlink: convertToString(uint64(qos.DnnMbrDownlink)),
					Uplink:   convertToString(uint64(qos.DnnMbrUplink)),
				},
				Var5gQosProfile: &models.SubscribedDefaultQos{
					Var5qi: 9,
					Arp: &models.Arp{
						PriorityLevel: 8,
					},
					PriorityLevel: 8,
				},
			},
		},
	}
	smDataBsonA := configmodels.ToBsonM(smData)
	smDataBsonA["ueId"] = "imsi-" + imsi
	smDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(smDataColl, filter, smDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update SM provisioned Data for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("updated SM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmfSelectionProvisionedData(snssai *models.Snssai, mcc, mnc, dnn, imsi string) error {
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			SnssaiModelsToHex(*snssai): {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: dnn,
					},
				},
			},
		},
	}
	smfSelecDataBsonA := configmodels.ToBsonM(smfSelData)
	smfSelecDataBsonA["ueId"] = "imsi-" + imsi
	smfSelecDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(smfSelDataColl, filter, smfSelecDataBsonA)
	if err != nil {
		logger.DbLog.Errorf("failed to update SMF selection provisioned data for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("updated SMF selection provisioned data for IMSI %s", imsi)
	return nil
}

func SnssaiModelsToHex(snssai models.Snssai) string {
	sst := fmt.Sprintf("%02x", snssai.Sst)
	return sst + snssai.Sd
}

func convertToString(val uint64) string {
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
			logger.DbLog.Errorf("could not unmarshall slice %v", rawSlice)
		}
		slices = append(slices, &sliceData)
	}
	return slices
}

func getSliceByName(name string) configmodels.Slice {
	filter := bson.M{"slice-name": name}
	sliceDataInterface, errGetOne := dbadapter.CommonDBClient.RestfulAPIGetOne(sliceDataColl, filter)
	if errGetOne != nil {
		logger.DbLog.Warnln(errGetOne)
	}
	var sliceData configmodels.Slice
	err := json.Unmarshal(configmodels.MapToByte(sliceDataInterface), &sliceData)
	if err != nil {
		logger.DbLog.Errorf("could not unmarshall slice %v", sliceDataInterface)
	}
	return sliceData
}

func handleNetworkSliceDelete(sliceName string) error {
	rwLock.Lock()
	prevSlice := getSliceByName(sliceName)
	defer rwLock.Unlock()
	filter := bson.M{"slice-name": sliceName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(sliceDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("failed to delete slice data for %v: %v", sliceName, err)
		return err
	}
	slice := getSliceByName(sliceName)
	if err := syncSliceDeviceGroupSubscribers(&slice, &prevSlice); err != nil {
		logger.WebUILog.Errorf("failed to sync slice %v: %v", sliceName, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to delete slice data for %v", sliceName)
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err := sendPebbleNotification("aetherproject.org/webconsole/networkslice/delete")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return nil
}

func getDeletedDeviceGroupsList(slice, prevSlice *configmodels.Slice) (names []string) {
	for prevSlice == nil {
		return
	}

	if slice != nil {
		for _, pdgName := range prevSlice.SiteDeviceGroup {
			var found bool
			for _, dgName := range slice.SiteDeviceGroup {
				if dgName == pdgName {
					found = true
					break
				}
			}
			if !found {
				names = append(names, pdgName)
			}
		}
	} else {
		names = append(names, prevSlice.SiteDeviceGroup...)
	}

	return
}
