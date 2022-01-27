// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0
package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/free5gc/MongoDBLibrary"
	"github.com/free5gc/openapi/models"
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
)

var configLog *logrus.Entry

func init() {
	configLog = logger.ConfigLog
}

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
var slicesConfigSnapshot map[string]*configmodels.Slice
var devgroupsConfigSnapshot map[string]*configmodels.DeviceGroups
var rwLock sync.RWMutex

func init() {
	slicesConfigSnapshot = make(map[string]*configmodels.Slice)
	devgroupsConfigSnapshot = make(map[string]*configmodels.DeviceGroups)
}

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
	firstConfigRcvd := false
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

					if factory.WebUIConfig.Configuration.Mode5G == true {
						var config5gMsg Update5GSubscriberMsg
						config5gMsg.Msg = configMsg
						config5gMsg.PrevDevGroup = devgroupsConfigSnapshot[configMsg.DevGroupName]
						subsUpdateChan <- &config5gMsg
					}
					rwLock.Lock()
					devgroupsConfigSnapshot[configMsg.DevGroupName] = configMsg.DevGroup
					rwLock.Unlock()
				}

				if configMsg.Slice != nil {
					configLog.Infof("Received Slice [%v] configuration from config channel", configMsg.SliceName)

					if factory.WebUIConfig.Configuration.Mode5G == true {
						var config5gMsg Update5GSubscriberMsg
						config5gMsg.Msg = configMsg
						config5gMsg.PrevSlice = slicesConfigSnapshot[configMsg.SliceName]
						subsUpdateChan <- &config5gMsg
					}
					rwLock.Lock()
					slicesConfigSnapshot[configMsg.SliceName] = configMsg.Slice
					rwLock.Unlock()
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
						config5gMsg.PrevDevGroup = devgroupsConfigSnapshot[configMsg.DevGroupName]
						delete(devgroupsConfigSnapshot, configMsg.DevGroupName)
					}

					if configMsg.Slice == nil {
						configLog.Infof("Received delete Slice [%v] from config channel", configMsg.SliceName)
						config5gMsg.PrevSlice = slicesConfigSnapshot[configMsg.SliceName]
						delete(slicesConfigSnapshot, configMsg.SliceName)
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

func isImsiPartofDeviceGroup(imsi string) bool {
	for _, devgroup := range devgroupsConfigSnapshot {
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
	MongoDBLibrary.RestfulAPIPost(amPolicyDataColl, filter, amPolicyDatBsonA)
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
	MongoDBLibrary.RestfulAPIPost(smPolicyDataColl, filter, smPolicyDatBsonA)
}

func updateAmProviosionedData(snssai *models.Snssai, mcc, mnc, dnn, imsi string) {
	amData := models.AccessAndMobilitySubscriptionData{
		Gpsis: []string{
			"msisdn-0900000000",
		},
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{*snssai},
			SingleNssais:        []models.Snssai{*snssai},
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: "2 Gbps",
			Uplink:   "1 Gbps",
		},
	}
	amDataBsonA := toBsonM(amData)
	amDataBsonA["ueId"] = "imsi-" + imsi
	amDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	MongoDBLibrary.RestfulAPIPost(amDataColl, filter, amDataBsonA)
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
	MongoDBLibrary.RestfulAPIPost(smDataColl, filter, smDataBsonA)
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
	MongoDBLibrary.RestfulAPIPost(smfSelDataColl, filter, smfSelecDataBsonA)
}

func isDeviceGroupExistInSlice(msg *Update5GSubscriberMsg) *configmodels.Slice {
	for name, slice := range slicesConfigSnapshot {
		for _, dgName := range slice.SiteDeviceGroup {
			if dgName == msg.Msg.DevGroupName {
				logger.WebUILog.Infof("Device Group [%v] is part of slice: %v", dgName, name)
				return slice
			}
		}
	}

	return nil
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
			logger.WebUILog.Debugln("Insert/Update AuthenticationSubscription")
			//check this Imsi is part of any of the devicegroup
			imsi := strings.ReplaceAll(confData.Msg.Imsi, "imsi-", "")
			if confData.Msg.MsgMethod != configmodels.Delete_op {
				filter := bson.M{"ueId": confData.Msg.Imsi}
				authDataBsonA := toBsonM(confData.Msg.AuthSubData)
				authDataBsonA["ueId"] = confData.Msg.Imsi
				MongoDBLibrary.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
			} else {
				filter := bson.M{"ueId": "imsi-" + imsi}
				MongoDBLibrary.RestfulAPIDeleteOne(authSubsDataColl, filter)
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
					updateAmProviosionedData(snssai, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
					updateSmProviosionedData(snssai, confData.Msg.DevGroup.IpDomainExpanded.UeDnnQos, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
					updateSmfSelectionProviosionedData(snssai, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi)
				}

				dimsis := getDeletedImsisList(confData.Msg.DevGroup, confData.PrevDevGroup)
				for _, imsi := range dimsis {
					mcc := slice.SiteInfo.Plmn.Mcc
					mnc := slice.SiteInfo.Plmn.Mnc
					filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
					filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
					MongoDBLibrary.RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
					MongoDBLibrary.RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
					MongoDBLibrary.RestfulAPIDeleteOne(amDataColl, filter)
					MongoDBLibrary.RestfulAPIDeleteOne(smDataColl, filter)
					MongoDBLibrary.RestfulAPIDeleteOne(smfSelDataColl, filter)
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
					devGroupConfig := devgroupsConfigSnapshot[dgName]
					if devgroupsConfigSnapshot[dgName] != nil {
						for _, imsi := range devGroupConfig.Imsis {
							dnn := devGroupConfig.IpDomainExpanded.Dnn
							mcc := slice.SiteInfo.Plmn.Mcc
							mnc := slice.SiteInfo.Plmn.Mnc
							updateAmPolicyData(imsi)
							updateSmPolicyData(snssai, dnn, imsi)
							updateAmProviosionedData(snssai, mcc, mnc, dnn, imsi)
							updateSmProviosionedData(snssai, devGroupConfig.IpDomainExpanded.UeDnnQos, mcc, mnc, dnn, imsi)
							updateSmfSelectionProviosionedData(snssai, mcc, mnc, dnn, imsi)
						}
					}
				}
			}

			dgnames := getDeleteGroupsList(slice, confData.PrevSlice)
			for _, dgname := range dgnames {
				if devgroupsConfigSnapshot[dgname] != nil {
					for _, imsi := range devgroupsConfigSnapshot[dgname].Imsis {
						mcc := confData.PrevSlice.SiteInfo.Plmn.Mcc
						mnc := confData.PrevSlice.SiteInfo.Plmn.Mnc
						filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
						filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
						MongoDBLibrary.RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
						MongoDBLibrary.RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
						MongoDBLibrary.RestfulAPIDeleteOne(amDataColl, filter)
						MongoDBLibrary.RestfulAPIDeleteOne(smDataColl, filter)
						MongoDBLibrary.RestfulAPIDeleteOne(smfSelDataColl, filter)
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
	kbVal = val / 1024
	mbVal = val / 1048576
	gbVal = val / 1073741824
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

	fmt.Println("convertToString value ", val, retStr)
	return retStr
}

// seems something which we should move to mongolib
func toBsonM(data interface{}) (ret bson.M) {
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &ret)
	return
}

func SnssaiModelsToHex(snssai models.Snssai) string {
	sst := fmt.Sprintf("%02x", snssai.Sst)
	return sst + snssai.Sd
}
