// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/omec-project/MongoDBLibrary"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	authSubsDataColl = "subscriptionData.authenticationData.authenticationSubscription"
	amDataColl       = "subscriptionData.provisionedData.amData"
	smDataColl       = "subscriptionData.provisionedData.smData"
	smfSelDataColl   = "subscriptionData.provisionedData.smfSelectionSubscriptionData"
	amPolicyDataColl = "policyData.ues.amData"
	smPolicyDataColl = "policyData.ues.smData"
	flowRuleDataColl = "policyData.ues.flowRule"
	devGroupDataColl = "webconsoleData.snapshots.devGroupData"
	sliceDataColl    = "webconsoleData.snapshots.sliceData"
)

var configLog *logrus.Entry

func init() {
	configLog = logger.ConfigLog
}

var (
	RestfulAPIGetMany   = MongoDBLibrary.RestfulAPIGetMany
	RestfulAPIGetOne    = MongoDBLibrary.RestfulAPIGetOne
	RestfulAPIPost      = MongoDBLibrary.RestfulAPIPost
	RestfulAPIDeleteOne = MongoDBLibrary.RestfulAPIDeleteOne
)

/*type SubsUpdMsg struct {
	UeIds         []string
	Nssai         models.Snssai
	ServingPlmnId string
	Qos           configmodels.SliceQos
}*/

type Update5GSubscriberMsg struct {
	Msg          *configmodels.ConfigMessage
	PrevDevGroup *configmodels.DeviceGroups
	PrevSlice    *configmodels.Slice
}

var subsChannel chan *Update5GSubscriberMsg
var rwLock sync.RWMutex

var initialConfigRcvd bool
var imsiData map[string]*models.AuthenticationSubscription

func init() {
	imsiData = make(map[string]*models.AuthenticationSubscription)
}

func configHandler(configMsgChan chan *configmodels.ConfigMessage, configReceived chan bool) {

	// Start Goroutine which will listens for subscriber config updates
	// and update the mongoDB. Only for 5G
	subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
	subsChannel = subsUpdateChan
	if factory.WebUIConfig.Configuration.Mode5G == true {
		go Config5GUpdateHandle(subsUpdateChan)
	}
	firstConfigRcvd := firstConfigReceived()
	if firstConfigRcvd {
		configReceived <- true
	}
	for {
		configLog.Infoln("Waiting for configuration event ")
		select {
		case configMsg := <-configMsgChan:
			//configLog.Infof("Received configuration event %v ", configMsg)
			if configMsg.MsgType == configmodels.Sub_data {
				imsiVal := strings.ReplaceAll(configMsg.Imsi, "imsi-", "")
				configLog.Infoln("Received imsi from config channel: ", imsiVal)
				rwLock.Lock()
				imsiData[imsiVal] = configMsg.AuthSubData
				rwLock.Unlock()
				configLog.Infof("Received Imsi [%v] configuration from config channel", configMsg.Imsi)
				handleSubscriberPost(configMsg)
				if factory.WebUIConfig.Configuration.Mode5G == true {
					var configUMsg Update5GSubscriberMsg
					configUMsg.Msg = configMsg
					subsUpdateChan <- &configUMsg
				}
			}

			if configMsg.MsgMethod == configmodels.Post_op || configMsg.MsgMethod == configmodels.Put_op {

				if firstConfigRcvd == false && (configMsg.MsgType == configmodels.Device_group || configMsg.MsgType == configmodels.Network_slice) {
					configLog.Debugln("First config received from ROC")
					firstConfigRcvd = true
					configReceived <- true
				}

				//configLog.Infoln("Received msg from configApi package ", configMsg)
				// update config snapshot
				if configMsg.DevGroup != nil {
					configLog.Infof("Received Device Group [%v] configuration from config channel", configMsg.DevGroupName)
					handleDeviceGroupPost(configMsg, subsUpdateChan)
				}

				if configMsg.Slice != nil {
					configLog.Infof("Received Slice [%v] configuration from config channel", configMsg.SliceName)
					handleNetworkSlicePost(configMsg, subsUpdateChan)
				}

				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
					configLog.Infoln("Push config for client : ", client.id)
					client.outStandingPushConfig <- configMsg
				}
			} else {
				var config5gMsg Update5GSubscriberMsg
				if configMsg.MsgType != configmodels.Sub_data {
					rwLock.Lock()
					// update config snapshot
					if configMsg.DevGroup == nil {
						configLog.Infof("Received delete Device Group [%v] from config channel", configMsg.DevGroupName)
						config5gMsg.PrevDevGroup = getDeviceGroupByName(configMsg.DevGroupName)
						filter := bson.M{"group-name": configMsg.DevGroupName}
						RestfulAPIDeleteOne(devGroupDataColl, filter)
					}

					if configMsg.Slice == nil {
						configLog.Infof("Received delete Slice [%v] from config channel", configMsg.SliceName)
						config5gMsg.PrevSlice = getSliceByName(configMsg.SliceName)
						filter := bson.M{"SliceName": configMsg.SliceName}
						RestfulAPIDeleteOne(sliceDataColl, filter)
					}
					rwLock.Unlock()
				} else {
					configLog.Infof("Received delete Subscriber [%v] from config channel", configMsg.Imsi)
				}
				if factory.WebUIConfig.Configuration.Mode5G == true {
					config5gMsg.Msg = configMsg
					subsUpdateChan <- &config5gMsg
				}
				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
					configLog.Infoln("Push config for client : ", client.id)
					client.outStandingPushConfig <- configMsg
				}
			}
		}
	}
}

func handleSubscriberPost(configMsg *configmodels.ConfigMessage) {
	rwLock.Lock()
	basicAmData := map[string]interface{}{
        "ueId": configMsg.Imsi,
    }
	filter := bson.M{"ueId": configMsg.Imsi}
	basicDataBson := toBsonM(basicAmData)
	RestfulAPIPost(amDataColl, filter, basicDataBson)
	rwLock.Unlock()
}

func handleDeviceGroupPost(configMsg *configmodels.ConfigMessage, subsUpdateChan chan *Update5GSubscriberMsg) {
	rwLock.Lock()
	if factory.WebUIConfig.Configuration.Mode5G == true {
		var config5gMsg Update5GSubscriberMsg
		config5gMsg.Msg = configMsg
		config5gMsg.PrevDevGroup = getDeviceGroupByName(configMsg.DevGroupName)
		subsUpdateChan <- &config5gMsg
	}
	filter := bson.M{"group-name": configMsg.DevGroupName}
	devGroupDataBsonA := toBsonM(configMsg.DevGroup)
	RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
	rwLock.Unlock()
}

func handleNetworkSlicePost(configMsg *configmodels.ConfigMessage, subsUpdateChan chan *Update5GSubscriberMsg) {
	rwLock.Lock()
	if factory.WebUIConfig.Configuration.Mode5G == true {
		var config5gMsg Update5GSubscriberMsg
		config5gMsg.Msg = configMsg
		config5gMsg.PrevSlice = getSliceByName(configMsg.SliceName)
		subsUpdateChan <- &config5gMsg
	}
	filter := bson.M{"SliceName": configMsg.SliceName}
	sliceDataBsonA := toBsonM(configMsg.Slice)
	RestfulAPIPost(sliceDataColl, filter, sliceDataBsonA)
	rwLock.Unlock()
}

func firstConfigReceived() bool {
	if len(getDeviceGroups()) > 0 {
		return true
	} else if len(getSlices()) > 0 {
		return true
	}
	return false
}

func getDeviceGroups() []*configmodels.DeviceGroups {
	rawDeviceGroups := RestfulAPIGetMany(devGroupDataColl, nil)
	var deviceGroups []*configmodels.DeviceGroups
	for _, rawDevGroup := range rawDeviceGroups {
		var devGroupData configmodels.DeviceGroups
		json.Unmarshal(mapToByte(rawDevGroup), &devGroupData)
		deviceGroups = append(deviceGroups, &devGroupData)
	}
	return deviceGroups
}

func getDeviceGroupByName(name string) *configmodels.DeviceGroups {
	filter := bson.M{"group-name": name}
	devGroupDataInterface := RestfulAPIGetOne(devGroupDataColl, filter)
	var devGroupData configmodels.DeviceGroups
	json.Unmarshal(mapToByte(devGroupDataInterface), &devGroupData)
	return &devGroupData
}

func getSlices() []*configmodels.Slice {
	rawSlices := RestfulAPIGetMany(sliceDataColl, nil)
	var slices []*configmodels.Slice
	for _, rawSlice := range rawSlices {
		var sliceData configmodels.Slice
		json.Unmarshal(mapToByte(rawSlice), &sliceData)
		slices = append(slices, &sliceData)
	}
	return slices
}

func getSliceByName(name string) *configmodels.Slice {
	filter := bson.M{"SliceName": name}
	sliceDataInterface := RestfulAPIGetOne(sliceDataColl, filter)
	var sliceData configmodels.Slice
	json.Unmarshal(mapToByte(sliceDataInterface), &sliceData)
	return &sliceData
}

func isImsiPartofDeviceGroup(imsi string) bool {
	for _, devgroup := range getDeviceGroups() {
		for _, val := range devgroup.Imsis {
			if val == imsi {
				return true
			}
		}
	}
	return false
}

func getAddedImsisList(group, prevGroup *configmodels.DeviceGroups) (aimsis []string) {
	if group == nil {
		return
	}
	for _, imsi := range group.Imsis {
		if prevGroup == nil {
			if imsiData[imsi] != nil {
				aimsis = append(aimsis, imsi)
			}
		} else {
			var found bool
			for _, pimsi := range prevGroup.Imsis {
				if pimsi == imsi {
					found = true
				}
			}

			if !found {
				aimsis = append(aimsis, imsi)
			}
		}
	}

	return
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

func updateAmPolicyData(imsi string) {
	//ampolicydata
	var amPolicy models.AmPolicyData
	amPolicy.SubscCats = append(amPolicy.SubscCats, "free5gc")
	amPolicyDatBsonA := toBsonM(amPolicy)
	amPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
	RestfulAPIPost(amPolicyDataColl, filter, amPolicyDatBsonA)
}

func updateSmPolicyData(snssai *models.Snssai, dnn string, imsi string) {

	var smPolicyData models.SmPolicyData
	var smPolicySnssaiData models.SmPolicySnssaiData
	dnnData := map[string]models.SmPolicyDnnData{
		dnn: {
			Dnn: dnn,
		},
	}
	//smpolicydata
	smPolicySnssaiData.Snssai = snssai
	smPolicySnssaiData.SmPolicyDnnData = dnnData
	smPolicyData.SmPolicySnssaiData = make(map[string]models.SmPolicySnssaiData)
	smPolicyData.SmPolicySnssaiData[SnssaiModelsToHex(*snssai)] = smPolicySnssaiData
	smPolicyDatBsonA := toBsonM(smPolicyData)
	smPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
	RestfulAPIPost(smPolicyDataColl, filter, smPolicyDatBsonA)
}

func updateAmProviosionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string) {
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
	amDataBsonA := toBsonM(amData)
	amDataBsonA["ueId"] = "imsi-" + imsi
	amDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{
		"ueId": "imsi-" + imsi,
		"$or": []bson.M{
			{"servingPlmnId": mcc + mnc},
			{"servingPlmnId": bson.M{"$exists": false}},
		},
	}
	RestfulAPIPost(amDataColl, filter, amDataBsonA)
}

func updateSmProviosionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string) {
	//TODO smData
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
	smDataBsonA := toBsonM(smData)
	smDataBsonA["ueId"] = "imsi-" + imsi
	smDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	RestfulAPIPost(smDataColl, filter, smDataBsonA)
}

func updateSmfSelectionProviosionedData(snssai *models.Snssai, mcc, mnc, dnn, imsi string) {
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			SnssaiModelsToHex(*snssai): {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: dnn,
					},
				},
			},
		}}
	smfSelecDataBsonA := toBsonM(smfSelData)
	smfSelecDataBsonA["ueId"] = "imsi-" + imsi
	smfSelecDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	RestfulAPIPost(smfSelDataColl, filter, smfSelecDataBsonA)
}

func isDeviceGroupExistInSlice(msg *Update5GSubscriberMsg) *configmodels.Slice {
	for name, slice := range getSlices() {
		for _, dgName := range slice.SiteDeviceGroup {
			if dgName == msg.Msg.DevGroupName {
				logger.WebUILog.Infof("Device Group [%v] is part of slice: %v", dgName, name)
				return slice
			}
		}
	}

	return nil
}

func getAddedGroupsList(slice, prevSlice *configmodels.Slice) (names []string) {
	return getDeleteGroupsList(prevSlice, slice)
}

func getDeleteGroupsList(slice, prevSlice *configmodels.Slice) (names []string) {
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
		for _, pdgName := range prevSlice.SiteDeviceGroup {
			names = append(names, pdgName)
		}
	}

	return
}

func Config5GUpdateHandle(confChan chan *Update5GSubscriberMsg) {
	for confData := range confChan {
		switch confData.Msg.MsgType {
		case configmodels.Sub_data:
			rwLock.RLock()
			//check this Imsi is part of any of the devicegroup
			imsi := strings.ReplaceAll(confData.Msg.Imsi, "imsi-", "")
			if confData.Msg.MsgMethod != configmodels.Delete_op {
				logger.WebUILog.Debugln("Insert/Update AuthenticationSubscription ", imsi)
				filter := bson.M{"ueId": confData.Msg.Imsi}
				authDataBsonA := toBsonM(confData.Msg.AuthSubData)
				authDataBsonA["ueId"] = confData.Msg.Imsi
				RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
			} else {
				logger.WebUILog.Debugln("Delete AuthenticationSubscription", imsi)
				filter := bson.M{"ueId": "imsi-" + imsi}
				RestfulAPIDeleteOne(authSubsDataColl, filter)
				RestfulAPIDeleteOne(amDataColl, filter)
			}
			rwLock.RUnlock()

		case configmodels.Device_group:
			rwLock.RLock()
			/* is this devicegroup part of any existing slice */
			slice := isDeviceGroupExistInSlice(confData)
			if slice != nil {
				sVal, _ := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
				snssai := &models.Snssai{
					Sd:  slice.SliceId.Sd,
					Sst: int32(sVal),
				}

				aimsis := getAddedImsisList(confData.Msg.DevGroup, confData.PrevDevGroup)
				for _, imsi := range aimsis {
					dnn := confData.Msg.DevGroup.IpDomainExpanded.Dnn
					updateAmPolicyData(imsi)
					updateSmPolicyData(snssai, dnn, imsi)
					updateAmProviosionedData(snssai, confData.Msg.DevGroup.IpDomainExpanded.UeDnnQos, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
					updateSmProviosionedData(snssai, confData.Msg.DevGroup.IpDomainExpanded.UeDnnQos, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
					updateSmfSelectionProviosionedData(snssai, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
				}

				dimsis := getDeletedImsisList(confData.Msg.DevGroup, confData.PrevDevGroup)
				for _, imsi := range dimsis {
					mcc := slice.SiteInfo.Plmn.Mcc
					mnc := slice.SiteInfo.Plmn.Mnc
					filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
					filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
					RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
					RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
					RestfulAPIDeleteOne(amDataColl, filter)
					RestfulAPIDeleteOne(smDataColl, filter)
					RestfulAPIDeleteOne(smfSelDataColl, filter)
				}
			}
			rwLock.RUnlock()

		case configmodels.Network_slice:
			rwLock.RLock()
			logger.WebUILog.Debugln("Insert/Update Network Slice")
			slice := confData.Msg.Slice
			if slice == nil && confData.PrevSlice != nil {
				logger.WebUILog.Debugln("Deleted Slice: ", confData.PrevSlice)
			}
			if slice != nil {
				sVal, _ := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
				snssai := &models.Snssai{
					Sd:  slice.SliceId.Sd,
					Sst: int32(sVal),
				}
				for _, dgName := range slice.SiteDeviceGroup {
					configLog.Infoln("dgName : ", dgName)
					devGroupConfig := getDeviceGroupByName(dgName)
					if devGroupConfig != nil {
						for _, imsi := range devGroupConfig.Imsis {
							dnn := devGroupConfig.IpDomainExpanded.Dnn
							mcc := slice.SiteInfo.Plmn.Mcc
							mnc := slice.SiteInfo.Plmn.Mnc
							updateAmPolicyData(imsi)
							updateSmPolicyData(snssai, dnn, imsi)
							updateAmProviosionedData(snssai, devGroupConfig.IpDomainExpanded.UeDnnQos, mcc, mnc, dnn, imsi)
							updateSmProviosionedData(snssai, devGroupConfig.IpDomainExpanded.UeDnnQos, mcc, mnc, dnn, imsi)
							updateSmfSelectionProviosionedData(snssai, mcc, mnc, dnn, imsi)
						}
					}
				}
			}

			dgnames := getDeleteGroupsList(slice, confData.PrevSlice)
			for _, dgname := range dgnames {
				devGroupConfig := getDeviceGroupByName(dgname)
				if devGroupConfig != nil {
					for _, imsi := range devGroupConfig.Imsis {
						mcc := confData.PrevSlice.SiteInfo.Plmn.Mcc
						mnc := confData.PrevSlice.SiteInfo.Plmn.Mnc
						filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
						filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
						RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
						RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
						RestfulAPIDeleteOne(amDataColl, filter)
						RestfulAPIDeleteOne(smDataColl, filter)
						RestfulAPIDeleteOne(smfSelDataColl, filter)
					}

				}
			}
			rwLock.RUnlock()
		}
	} //end of for loop

}

func sliceToByte(data []map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}

func compareNssai(sNssai *models.Snssai,
	sliceId *models.Snssai) int {
	if sNssai.Sst != sliceId.Sst {
		return 1
	}
	return strings.Compare(sNssai.Sd, sliceId.Sd)
}

func convertToString(val uint64) string {
	var mbVal, gbVal, kbVal uint64
	kbVal = val / 1000
	mbVal = val / 1000000
	gbVal = val / 1000000000
	var retStr string
	if gbVal != 0 {
		retStr = strconv.FormatUint(uint64(gbVal), 10) + " Gbps"
	} else if mbVal != 0 {
		retStr = strconv.FormatUint(uint64(mbVal), 10) + " Mbps"
	} else if kbVal != 0 {
		retStr = strconv.FormatUint(uint64(kbVal), 10) + " Kbps"
	} else {
		retStr = strconv.FormatUint(uint64(val), 10) + " bps"
	}

	return retStr
}

// seems something which we should move to mongolib
func toBsonM(data interface{}) (ret bson.M) {
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &ret)
	return
}

func mapToByte(data map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}

func SnssaiModelsToHex(snssai models.Snssai) string {
	sst := fmt.Sprintf("%02x", snssai.Sst)
	return sst + snssai.Sd
}
