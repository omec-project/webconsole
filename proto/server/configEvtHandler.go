// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0
package server

import (
	"github.com/free5gc/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
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

// HandleSubscriberAdd : Update Info of subscriber
func HandleSubscriberAdd(imsiVal string) {
	var dgNameVal string
	imsiVal = strings.ReplaceAll(imsiVal, "imsi-", "")
	configLog.Infoln("UpdateInfo for UE : ", imsiVal)
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

func configHandler(configMsgChan chan *configmodels.ConfigMessage,
	subsChannelVal chan *SubsUpdMsg) {
	subsChannel = subsChannelVal
	for {
		configLog.Infoln("Waiting for configuration event ")
		select {
		case configMsg := <-configMsgChan:
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
					subsChannel <- &subsMsgData
				}

				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
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
