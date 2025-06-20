// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"sync"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configapi"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
)

const (
	authSubsDataColl = "subscriptionData.authenticationData.authenticationSubscription"
	amDataColl       = "subscriptionData.provisionedData.amData"
	devGroupDataColl = "webconsoleData.snapshots.devGroupData"
	sliceDataColl    = "webconsoleData.snapshots.sliceData"
)

type Update5GSubscriberMsg struct {
	Msg          *configmodels.ConfigMessage
	PrevDevGroup *configmodels.DeviceGroups
	PrevSlice    *configmodels.Slice
}

var (
	rwLock             sync.RWMutex
	subscriberAuthData configapi.SubscriberAuthenticationData
)

func configHandler(configMsgChan chan *configmodels.ConfigMessage, configReceived chan bool) {
	firstConfigRcvd := firstConfigReceived()
	if firstConfigRcvd {
		configReceived <- true
	}
	for {
		logger.ConfigLog.Infoln("waiting for configuration event")
		configMsg := <-configMsgChan

		if configMsg.MsgMethod == configmodels.Post_op || configMsg.MsgMethod == configmodels.Put_op {
			if !firstConfigRcvd && (configMsg.MsgType == configmodels.Device_group || configMsg.MsgType == configmodels.Network_slice) {
				logger.ConfigLog.Debugln("first config received from ROC")
				firstConfigRcvd = true
				configReceived <- true
			}
			if len(clientNFPool) == 0 {
				logger.ConfigLog.Infoln("no client available. No need to send config")
			}
			for _, client := range clientNFPool {
				logger.ConfigLog.Infoln("push config for client:", client.id)
				client.outStandingPushConfig <- configMsg
			}
		}
	}
}

func firstConfigReceived() bool {
	return len(getDeviceGroups()) > 0 || len(getSlices()) > 0
}

func getDeviceGroups() []*configmodels.DeviceGroups {
	rawDeviceGroups, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(devGroupDataColl, nil)
	if errGetMany != nil {
		logger.DbLog.Warnln(errGetMany)
	}
	var deviceGroups []*configmodels.DeviceGroups
	for _, rawDevGroup := range rawDeviceGroups {
		var devGroupData configmodels.DeviceGroups
		err := json.Unmarshal(configmodels.MapToByte(rawDevGroup), &devGroupData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshall device group %v", rawDevGroup)
		}
		deviceGroups = append(deviceGroups, &devGroupData)
	}
	return deviceGroups
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

func getAddedImsisList(group, prevGroup *configmodels.DeviceGroups) (aimsis []string) {
	if group == nil {
		return
	}
	for _, imsi := range group.Imsis {
		if prevGroup == nil {
			if subscriberAuthData.SubscriberAuthenticationDataGet("imsi-"+imsi) != nil {
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
		names = append(names, prevSlice.SiteDeviceGroup...)
	}

	return
}
