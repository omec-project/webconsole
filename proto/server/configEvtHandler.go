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

type Update5GSubscriberMsg struct {
	Msg          *configmodels.ConfigMessage
	PrevDevGroup *configmodels.DeviceGroups
	PrevSlice    *configmodels.Slice
}

var rwLock sync.RWMutex

var imsiData map[string]*models.AuthenticationSubscription

func init() {
	imsiData = make(map[string]*models.AuthenticationSubscription)
}

type MongoManyGetter interface {
	RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error)
}

type MongoOneGetter interface {
	RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error)
}

type MongoOneDeleterfromAuthDB interface {
	RestfulAPIDeleteOne(collName string, filter bson.M) error
}

type MongoOneDeleter interface {
	RestfulAPIDeleteOne(collName string, filter bson.M) error
}

type MongoPoster interface {
	RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error)
}

type MongoPosterforAuthDB interface {
	RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error)
}

func configHandler(configMsgChan chan *configmodels.ConfigMessage, configReceived chan bool, oneDeleter MongoOneDeleter, authDBOneDeleter MongoOneDeleterfromAuthDB, authDBPoster MongoPosterforAuthDB, manyGetter MongoManyGetter, poster MongoPoster, oneGetter MongoOneGetter) {

	// Start Goroutine which will listens for subscriber config updates
	// and update the mongoDB. Only for 5G
	subsUpdateChan := make(chan *Update5GSubscriberMsg, 10)
	if factory.WebUIConfig.Configuration.Mode5G == true {
		go Config5GUpdateHandle(subsUpdateChan, oneDeleter, authDBOneDeleter, authDBPoster, manyGetter, poster, oneGetter)
	}
	firstConfigRcvd := firstConfigReceived(manyGetter)
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
				handleSubscriberPost(configMsg, poster)
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
					handleDeviceGroupPost(configMsg, subsUpdateChan, poster, oneGetter)
				}

				if configMsg.Slice != nil {
					configLog.Infof("Received Slice [%v] configuration from config channel", configMsg.SliceName)
					handleNetworkSlicePost(configMsg, subsUpdateChan, poster, oneGetter)
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
						config5gMsg.PrevDevGroup = getDeviceGroupByName(configMsg.DevGroupName, oneGetter)
						filter := bson.M{"group-name": configMsg.DevGroupName}
						oneDeleter.RestfulAPIDeleteOne(devGroupDataColl, filter)
					}

					if configMsg.Slice == nil {
						configLog.Infof("Received delete Slice [%v] from config channel", configMsg.SliceName)
						config5gMsg.PrevSlice = getSliceByName(configMsg.SliceName, oneGetter)
						filter := bson.M{"SliceName": configMsg.SliceName}
						oneDeleter.RestfulAPIDeleteOne(sliceDataColl, filter)
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

func handleSubscriberPost(configMsg *configmodels.ConfigMessage, poster MongoPoster) {
	rwLock.Lock()
	basicAmData := map[string]interface{}{
		"ueId": configMsg.Imsi,
	}
	filter := bson.M{"ueId": configMsg.Imsi}
	basicDataBson := toBsonM(basicAmData)
	poster.RestfulAPIPost(amDataColl, filter, basicDataBson)
	rwLock.Unlock()
}

func handleDeviceGroupPost(configMsg *configmodels.ConfigMessage, subsUpdateChan chan *Update5GSubscriberMsg, poster MongoPoster, oneGetter MongoOneGetter) {
	rwLock.Lock()
	if factory.WebUIConfig.Configuration.Mode5G == true {
		var config5gMsg Update5GSubscriberMsg
		config5gMsg.Msg = configMsg
		config5gMsg.PrevDevGroup = getDeviceGroupByName(configMsg.DevGroupName, oneGetter)
		subsUpdateChan <- &config5gMsg
	}
	filter := bson.M{"group-name": configMsg.DevGroupName}
	devGroupDataBsonA := toBsonM(configMsg.DevGroup)
	poster.RestfulAPIPost(devGroupDataColl, filter, devGroupDataBsonA)
	rwLock.Unlock()
}

func handleNetworkSlicePost(configMsg *configmodels.ConfigMessage, subsUpdateChan chan *Update5GSubscriberMsg, poster MongoPoster, oneGetter MongoOneGetter) {
	rwLock.Lock()
	if factory.WebUIConfig.Configuration.Mode5G == true {
		var config5gMsg Update5GSubscriberMsg
		config5gMsg.Msg = configMsg
		config5gMsg.PrevSlice = getSliceByName(configMsg.SliceName, oneGetter)
		subsUpdateChan <- &config5gMsg
	}
	filter := bson.M{"SliceName": configMsg.SliceName}
	sliceDataBsonA := toBsonM(configMsg.Slice)
	poster.RestfulAPIPost(sliceDataColl, filter, sliceDataBsonA)
	rwLock.Unlock()
}

func firstConfigReceived(manyGetter MongoManyGetter) bool {
	return len(getDeviceGroups(manyGetter)) > 0 || len(getSlices(manyGetter)) > 0
}

func getDeviceGroups(manyGetter MongoManyGetter) []*configmodels.DeviceGroups {
	rawDeviceGroups, _ := manyGetter.RestfulAPIGetMany(devGroupDataColl, nil)
	var deviceGroups []*configmodels.DeviceGroups
	for _, rawDevGroup := range rawDeviceGroups {
		var devGroupData configmodels.DeviceGroups
		json.Unmarshal(mapToByte(rawDevGroup), &devGroupData)
		deviceGroups = append(deviceGroups, &devGroupData)
	}
	return deviceGroups
}

func getDeviceGroupByName(name string, oneGetter MongoOneGetter) *configmodels.DeviceGroups {
	filter := bson.M{"group-name": name}
	devGroupDataInterface, _ := oneGetter.RestfulAPIGetOne(devGroupDataColl, filter)
	var devGroupData configmodels.DeviceGroups
	json.Unmarshal(mapToByte(devGroupDataInterface), &devGroupData)
	return &devGroupData
}

func getSlices(manyGetter MongoManyGetter) []*configmodels.Slice {
	rawSlices, _ := manyGetter.RestfulAPIGetMany(sliceDataColl, nil)
	var slices []*configmodels.Slice
	for _, rawSlice := range rawSlices {
		var sliceData configmodels.Slice
		json.Unmarshal(mapToByte(rawSlice), &sliceData)
		slices = append(slices, &sliceData)
	}
	return slices
}

func getSliceByName(name string, oneGetter MongoOneGetter) *configmodels.Slice {
	filter := bson.M{"SliceName": name}
	sliceDataInterface, _ := oneGetter.RestfulAPIGetOne(sliceDataColl, filter)
	var sliceData configmodels.Slice
	json.Unmarshal(mapToByte(sliceDataInterface), &sliceData)
	return &sliceData
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

func updateAmPolicyData(imsi string, poster MongoPoster) {
	//ampolicydata
	var amPolicy models.AmPolicyData
	amPolicy.SubscCats = append(amPolicy.SubscCats, "free5gc")
	amPolicyDatBsonA := toBsonM(amPolicy)
	amPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
	poster.RestfulAPIPost(amPolicyDataColl, filter, amPolicyDatBsonA)
}

func updateSmPolicyData(snssai *models.Snssai, dnn string, imsi string, poster MongoPoster) {
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
	poster.RestfulAPIPost(smPolicyDataColl, filter, smPolicyDatBsonA)
}

func updateAmProviosionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string, poster MongoPoster) {
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
	poster.RestfulAPIPost(amDataColl, filter, amDataBsonA)
}

func updateSmProviosionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string, poster MongoPoster) {
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
	poster.RestfulAPIPost(smDataColl, filter, smDataBsonA)
}

func updateSmfSelectionProviosionedData(snssai *models.Snssai, mcc, mnc, dnn, imsi string, poster MongoPoster) {
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
	poster.RestfulAPIPost(smfSelDataColl, filter, smfSelecDataBsonA)
}

func isDeviceGroupExistInSlice(msg *Update5GSubscriberMsg, manyGetter MongoManyGetter) *configmodels.Slice {
	for name, slice := range getSlices(manyGetter) {
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

func Config5GUpdateHandle(confChan chan *Update5GSubscriberMsg, oneDeleter MongoOneDeleter, authDBOneDeleter MongoOneDeleterfromAuthDB, authDBPoster MongoPosterforAuthDB, manyGetter MongoManyGetter, poster MongoPoster, oneGetter MongoOneGetter) {
	for confData := range confChan {
		switch confData.Msg.MsgType {
		case configmodels.Sub_data:
			rwLock.RLock()
			// Check this IMSI is part of any device group
			imsi := strings.ReplaceAll(confData.Msg.Imsi, "imsi-", "")
			if confData.Msg.MsgMethod != configmodels.Delete_op {
				logger.WebUILog.Debugln("Insert/Update AuthenticationSubscription ", imsi)
				filter := bson.M{"ueId": confData.Msg.Imsi}
				authDataBsonA := toBsonM(confData.Msg.AuthSubData)
				authDataBsonA["ueId"] = confData.Msg.Imsi
				authDBPoster.RestfulAPIPost(authSubsDataColl, filter, authDataBsonA)
			} else {
				logger.WebUILog.Debugln("Delete AuthenticationSubscription", imsi)
				filter := bson.M{"ueId": "imsi-" + imsi}
				authDBOneDeleter.RestfulAPIDeleteOne(authSubsDataColl, filter)
				oneDeleter.RestfulAPIDeleteOne(amDataColl, filter)
			}
			rwLock.RUnlock()

		case configmodels.Device_group:
			rwLock.RLock()
			/* is this devicegroup part of any existing slice */
			slice := isDeviceGroupExistInSlice(confData, manyGetter)
			if slice != nil {
				sVal, _ := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
				snssai := &models.Snssai{
					Sd:  slice.SliceId.Sd,
					Sst: int32(sVal),
				}

				aimsis := getAddedImsisList(confData.Msg.DevGroup, confData.PrevDevGroup)
				for _, imsi := range aimsis {
					dnn := confData.Msg.DevGroup.IpDomainExpanded.Dnn
					updateAmPolicyData(imsi, poster)
					updateSmPolicyData(snssai, dnn, imsi, poster)
					updateAmProviosionedData(snssai, confData.Msg.DevGroup.IpDomainExpanded.UeDnnQos, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi, poster)
					updateSmProviosionedData(snssai, confData.Msg.DevGroup.IpDomainExpanded.UeDnnQos, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi, poster)
					updateSmfSelectionProviosionedData(snssai, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, dnn, imsi, poster)
				}

				dimsis := getDeletedImsisList(confData.Msg.DevGroup, confData.PrevDevGroup)
				for _, imsi := range dimsis {
					mcc := slice.SiteInfo.Plmn.Mcc
					mnc := slice.SiteInfo.Plmn.Mnc
					filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
					filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
					oneDeleter.RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
					oneDeleter.RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
					oneDeleter.RestfulAPIDeleteOne(amDataColl, filter)
					oneDeleter.RestfulAPIDeleteOne(smDataColl, filter)
					oneDeleter.RestfulAPIDeleteOne(smfSelDataColl, filter)
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
					devGroupConfig := getDeviceGroupByName(dgName, oneGetter)
					if devGroupConfig != nil {
						for _, imsi := range devGroupConfig.Imsis {
							dnn := devGroupConfig.IpDomainExpanded.Dnn
							mcc := slice.SiteInfo.Plmn.Mcc
							mnc := slice.SiteInfo.Plmn.Mnc
							updateAmPolicyData(imsi, poster)
							updateSmPolicyData(snssai, dnn, imsi, poster)
							updateAmProviosionedData(snssai, devGroupConfig.IpDomainExpanded.UeDnnQos, mcc, mnc, dnn, imsi, poster)
							updateSmProviosionedData(snssai, devGroupConfig.IpDomainExpanded.UeDnnQos, mcc, mnc, dnn, imsi, poster)
							updateSmfSelectionProviosionedData(snssai, mcc, mnc, dnn, imsi, poster)
						}
					}
				}
			}

			dgnames := getDeleteGroupsList(slice, confData.PrevSlice)
			for _, dgname := range dgnames {
				devGroupConfig := getDeviceGroupByName(dgname, oneGetter)
				if devGroupConfig != nil {
					for _, imsi := range devGroupConfig.Imsis {
						mcc := confData.PrevSlice.SiteInfo.Plmn.Mcc
						mnc := confData.PrevSlice.SiteInfo.Plmn.Mnc
						filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
						filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
						oneDeleter.RestfulAPIDeleteOne(amPolicyDataColl, filterImsiOnly)
						oneDeleter.RestfulAPIDeleteOne(smPolicyDataColl, filterImsiOnly)
						oneDeleter.RestfulAPIDeleteOne(amDataColl, filter)
						oneDeleter.RestfulAPIDeleteOne(smDataColl, filter)
						oneDeleter.RestfulAPIDeleteOne(smfSelDataColl, filter)
					}
				}
			}
			rwLock.RUnlock()
		}
	} //end of for loop
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
