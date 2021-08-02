// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0
package server

import (
	"encoding/json"
	"github.com/free5gc/MongoDBLibrary"
	"github.com/free5gc/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"strconv"
	"strings"
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

type SubsUpdMsg struct {
	UeIds         []string
	Nssai         models.Snssai
	ServingPlmnId string
	Qos           configmodels.SliceQos
}

var subsChannel chan *SubsUpdMsg
var slicesConfigSnapshot map[string]*configmodels.Slice
var devgroupsConfigSnapshot map[string]*configmodels.DeviceGroups

func init() {
	slicesConfigSnapshot = make(map[string]*configmodels.Slice)
	devgroupsConfigSnapshot = make(map[string]*configmodels.DeviceGroups)
}

var initialConfigRcvd bool
var imsiData map[string]*models.AuthenticationSubscription

func init() {
	imsiData = make(map[string]*models.AuthenticationSubscription)
}

// HandleSubscriberAdd : Update Info of subscriber
func HandleSubscriberAdd(imsiVal string, authSubsData *models.AuthenticationSubscription) {
	var dgNameVal string
	imsiVal = strings.ReplaceAll(imsiVal, "imsi-", "")
	configLog.Infoln("UpdateInfo for UE : ", imsiVal)
	configLog.Infoln("Device Group snapshot ", devgroupsConfigSnapshot)
	imsiData[imsiVal] = authSubsData
	for key, dvcGrp := range devgroupsConfigSnapshot {
		for _, imsi := range dvcGrp.Imsis {
			if strings.Compare(imsi, imsiVal) == 0 {
				dgNameVal = key
				break
			}
		}
		if dgNameVal == "" {
			continue
		}

		configLog.Infoln("added imsi in map ", imsiData)
		for _, slice := range slicesConfigSnapshot {
			var subsMsgData SubsUpdMsg
			subsMsgData.UeIds = nil
			for _, dgName := range slice.SiteDeviceGroup {
				configLog.Infoln("dgName : ", dgName)
				if strings.Compare(dgName, dgNameVal) == 0 {
					sVal, err :=
						strconv.ParseUint(slice.SliceId.Sst,
							10, 32)
					if err != nil {
						sVal = 0
					}
					subsMsgData.Nssai.Sst = int32(sVal)
					subsMsgData.Nssai.Sd = slice.SliceId.Sd
					subsMsgData.ServingPlmnId = slice.SiteInfo.Plmn.Mcc + slice.SiteInfo.Plmn.Mnc
					subsMsgData.Qos = slice.Qos
					var ueID string = "imsi-" + imsiVal
					configLog.Infoln("ueID : ", ueID)
					subsMsgData.UeIds = append(subsMsgData.UeIds, ueID)
					configLog.Infoln("len of UeIds : ", len(subsMsgData.UeIds))
					subsChannel <- &subsMsgData
					break
				}
			}
		}
	}
}

func configHandler(configMsgChan chan *configmodels.ConfigMessage, configReceived chan bool) {

	// Start Goroutine which will listens for subscriber config updates
	// and update the mongoDB. Only for 5G
	subsUpdateChan := make(chan *SubsUpdMsg, 10)
	if factory.WebUIConfig.Configuration.Mode5G == true {
		go SubscriptionUpdateHandle(subsUpdateChan)
	}
	firstConfigRcvd := false
	for {
		configLog.Infoln("Waiting for configuration event ")
		select {
		case configMsg := <-configMsgChan:

			if firstConfigRcvd == false {
				firstConfigRcvd = true
				configReceived <- true
			}

			if configMsg.MsgMethod == configmodels.Post_op || configMsg.MsgMethod == configmodels.Put_op {
				configLog.Infoln("Received msg from configApi package ", configMsg)
				// update config snapshot
				if configMsg.DevGroup != nil {
					configLog.Infoln("Received msg from configApi package for Device Group ", configMsg.DevGroupName)
					devgroupsConfigSnapshot[configMsg.DevGroupName] = configMsg.DevGroup
				}

				if configMsg.Slice != nil {
					configLog.Infoln("Received msg from configApi package for Slice ", configMsg.SliceName)
					slicesConfigSnapshot[configMsg.SliceName] = configMsg.Slice
				}

				if factory.WebUIConfig.Configuration.Mode5G == true {
					for _, slice := range slicesConfigSnapshot {
						var subsMsgData SubsUpdMsg
						sVal, err :=
							strconv.ParseUint(slice.SliceId.Sst,
								10, 32)
						if err != nil {
							sVal = 0
						}
						subsMsgData.Nssai.Sst = int32(sVal)
						subsMsgData.Nssai.Sd =
							slice.SliceId.Sd
						subsMsgData.ServingPlmnId = slice.SiteInfo.Plmn.Mcc + slice.SiteInfo.Plmn.Mnc
						subsMsgData.Qos = slice.Qos
						subsMsgData.UeIds = nil
						for _, dgName := range slice.SiteDeviceGroup {
							configLog.Infoln("dgName : ", dgName)
							devGroupConfig := devgroupsConfigSnapshot[dgName]
							for _, imsi := range devGroupConfig.Imsis {
								var ueID string = "imsi-" + imsi
								configLog.Infoln("ueID : ", ueID)
								subsMsgData.UeIds =
									append(subsMsgData.UeIds, ueID)
							}
						}

						configLog.Infoln("len of UeIds : ", len(subsMsgData.UeIds))
						configLog.Infoln("slice sst : ", sVal, " sd: ", slice.SliceId.Sd)
						subsUpdateChan <- &subsMsgData
					}
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
				configLog.Infoln("Received delete msg from configApi package ", configMsg)
				// update config snapshot
				if configMsg.DevGroup == nil {
					configLog.Infoln("Received msg from configApi package to delete Device Group ", configMsg.DevGroupName)
					devgroupsConfigSnapshot[configMsg.DevGroupName] = nil
				}

				if configMsg.Slice == nil {
					configLog.Infoln("Received msg from configApi package to delete Slice ", configMsg.SliceName)
					slicesConfigSnapshot[configMsg.SliceName] = nil
				}

				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
					client.outStandingPushConfig <- configMsg
				}
			}
		}
	}
}

// SubscriptionUpdateHandle : Handle subscription update
func SubscriptionUpdateHandle(subsUpdateChan chan *SubsUpdMsg) {
	for subsData := range subsUpdateChan {
		logger.WebUILog.Infoln("SubscriptionUpdateHandle")
		var smDataData []models.SessionManagementSubscriptionData
		var smDatasBsonA []interface{}
		filterEmpty := bson.M{}
		var ueID string
		for _, ueID = range subsData.UeIds {
			filter := bson.M{"ueId": ueID}
			smDataDataInterface := MongoDBLibrary.RestfulAPIGetMany(smDataColl, filter)
			var found bool = false
			json.Unmarshal(sliceToByte(smDataDataInterface), &smDataData)
			if len(smDataData) != 0 {
				smDatasBsonA = make([]interface{}, 0, len(smDataData))
				for _, data := range smDataData {
					if compareNssai(data.SingleNssai, &subsData.Nssai) == 0 {
						logger.WebUILog.Infoln("entry exists for Imsi :  with SST:  and SD: ",
							ueID, subsData.Nssai.Sst, subsData.Nssai.Sd)
						found = true
						break
					}
				}

				if !found {
					logger.WebUILog.Infoln("entry doesnt exist for Imsi : %v with SST: %v and SD: %v",
						ueID, subsData.Nssai.Sst, subsData.Nssai.Sd)
					data := smDataData[0]
					data.SingleNssai.Sst = subsData.Nssai.Sst
					data.SingleNssai.Sd = subsData.Nssai.Sd
					data.SingleNssai.Sd = subsData.Nssai.Sd
					for idx, dnnCfg := range data.DnnConfigurations {
						var sessAmbr models.Ambr
						sessAmbr.Uplink = convertToString(uint32(subsData.Qos.Uplink))
						sessAmbr.Downlink = convertToString(uint32(subsData.Qos.Downlink))
						dnnCfg.SessionAmbr = &sessAmbr
						data.DnnConfigurations[idx] = dnnCfg
						logger.WebUILog.Infoln("uplink mbr ", data.DnnConfigurations[idx].SessionAmbr.Uplink)
						logger.WebUILog.Infoln("downlink mbr ", data.DnnConfigurations[idx].SessionAmbr.Downlink)
					}
					smDataBsonM := toBsonM(data)
					smDataBsonM["ueId"] = ueID
					smDataBsonM["servingPlmnId"] = subsData.ServingPlmnId
					logger.WebUILog.Infoln("servingplmnid ", subsData.ServingPlmnId)
					smDatasBsonA = append(smDatasBsonA, smDataBsonM)
				}
			} else {
				logger.WebUILog.Infoln("No imsi entry in db for imsi ", ueID)
			}
		}

		if len(smDatasBsonA) != 0 {
			MongoDBLibrary.RestfulAPIPostMany(smDataColl, filterEmpty, smDatasBsonA)
		}
	}
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

func convertToString(val uint32) string {
	var mbVal, gbVal, kbVal uint32
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

	return retStr
}

// seems something which we should move to mongolib
func toBsonM(data interface{}) (ret bson.M) {
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &ret)
	return
}
