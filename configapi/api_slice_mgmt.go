// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	KPS = 1000
	MPS = 1000000
	GPS = 1000000000
)

var configChannel chan *configmodels.ConfigMessage

func SetChannel(cfgChannel chan *configmodels.ConfigMessage) {
	logger.ConfigLog.Infoln("setting configChannel")
	configChannel = cfgChannel
}

func deviceGroupDeleteHelper(groupName string) error {
	logger.ConfigLog.Infof("received Delete Group %v request", groupName)
	if err := updateDeviceGroupInNetworkSlices(groupName); err != nil {
		return fmt.Errorf("error updating device group: %v in network slices: %v", groupName, err)
	}
	if err := handleDeviceGroupDelete(groupName); err != nil {
		return fmt.Errorf("error deleting device group %v: %v", groupName, err)
	}
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = configmodels.Delete_op
	msg.DevGroupName = groupName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Device Group [%v] with delete_op to config channel", groupName)
	return nil
}

func updateDeviceGroupInNetworkSlices(groupName string) error {
	filterByDeviceGroup := bson.M{"site-device-group": groupName}
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filterByDeviceGroup)
	if err != nil {
		logger.DbLog.Errorw("failed to retrieve network slices", "error", err)
		return err
	}
	var errorOccurred bool
	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			logger.DbLog.Errorf("could not unmarshal network slice %v", rawNetworkSlice)
			errorOccurred = true
			continue
		}
		prevSlice := getSliceByName(networkSlice.SliceName)
		networkSlice.SiteDeviceGroup = slices.DeleteFunc(networkSlice.SiteDeviceGroup, func(existingDG string) bool {
			return groupName == existingDG
		})
		if err = handleNetworkSlicePost(&networkSlice, &prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error posting slice %v: %v", networkSlice.SliceName, err)
			errorOccurred = true
			continue
		}
		msg := &configmodels.ConfigMessage{
			MsgMethod: configmodels.Post_op,
			MsgType:   configmodels.Network_slice,
			Slice:     &networkSlice,
			SliceName: networkSlice.SliceName,
		}
		configChannel <- msg
		logger.ConfigLog.Infof("network slice [%v] update sent to config channel", networkSlice.SliceName)
	}
	if errorOccurred {
		return fmt.Errorf("one or more network slice updates failed (see logs)")
	}
	return nil
}

func convertToBps(val int64, unit string) (bitrate int64) {
	if strings.EqualFold(unit, "bps") {
		bitrate = val
	} else if strings.EqualFold(unit, "kbps") {
		bitrate = val * KPS
	} else if strings.EqualFold(unit, "mbps") {
		bitrate = val * MPS
	} else if strings.EqualFold(unit, "gbps") {
		bitrate = val * GPS
	}
	// default consider it as bps
	return bitrate
}

func deviceGroupPostHelper(c *gin.Context, msgOp int, groupName string) error {
	logger.ConfigLog.Infof("received device group: %v", groupName)
	var request configmodels.DeviceGroups

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != "application/json" {
		err := fmt.Errorf("unsupported content-type: %s", ct)
		logger.ConfigLog.Errorln(err)
		return err
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ConfigLog.Errorf("JSON bind error: %v", err)
		return err
	}

	ipdomain := &request.IpDomainExpanded
	if ipdomain == nil {
		return fmt.Errorf("IpDomainExpanded is missing from device group payload")
	}

	logger.ConfigLog.Infof("imsis.size: %v, Imsis: %v", len(request.Imsis), request.Imsis)
	logger.ConfigLog.Infof("IP Domain Name: %v", request.IpDomainName)
	logger.ConfigLog.Infof("IP Domain details: %v", ipdomain)
	logger.ConfigLog.Infof("dnn name: %v", ipdomain.Dnn)
	logger.ConfigLog.Infof("ue pool: %v", ipdomain.UeIpPool)
	logger.ConfigLog.Infof("dns Primary: %v", ipdomain.DnsPrimary)
	logger.ConfigLog.Infof("dns Secondary: %v", ipdomain.DnsSecondary)
	logger.ConfigLog.Infof("ip mtu: %v", ipdomain.Mtu)
	logger.ConfigLog.Infof("device Group Name: %v", groupName)

	if ipdomain.UeDnnQos != nil {
		ipdomain.UeDnnQos.DnnMbrDownlink = convertToBps(ipdomain.UeDnnQos.DnnMbrDownlink, ipdomain.UeDnnQos.BitrateUnit)
		if ipdomain.UeDnnQos.DnnMbrDownlink < 0 {
			ipdomain.UeDnnQos.DnnMbrDownlink = math.MaxInt64
		}
		logger.ConfigLog.Infof("MbrDownLink: %v", ipdomain.UeDnnQos.DnnMbrDownlink)
		ipdomain.UeDnnQos.DnnMbrUplink = convertToBps(ipdomain.UeDnnQos.DnnMbrUplink, ipdomain.UeDnnQos.BitrateUnit)
		if ipdomain.UeDnnQos.DnnMbrUplink < 0 {
			ipdomain.UeDnnQos.DnnMbrUplink = math.MaxInt64
		}
		logger.ConfigLog.Infof("MbrUpLink: %v", ipdomain.UeDnnQos.DnnMbrUplink)
	}
	prevDevGroup := getDeviceGroupByName(groupName)
	if err := handleDeviceGroupPost(request, prevDevGroup); err != nil {
		logger.ConfigLog.Errorf("error posting device group %v: %v", request, err)
		return err
	}
	var msg configmodels.ConfigMessage
	request.DeviceGroupName = groupName
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = msgOp
	msg.DevGroup = &request
	msg.DevGroupName = groupName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully added Device Group [%v] to config channel", groupName)
	return nil
}

func networkSliceDeleteHelper(sliceName string) error {
	if err := handleNetworkSliceDelete(sliceName); err != nil {
		logger.ConfigLog.Errorf("Error deleting slice %v: %v", sliceName, err)
		return err
	}
	var msg configmodels.ConfigMessage
	msg.MsgMethod = configmodels.Delete_op
	msg.MsgType = configmodels.Network_slice
	msg.SliceName = sliceName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Network Slice [%v] with delete_op to config channel", sliceName)
	return nil
}

func networkSlicePostHelper(c *gin.Context, msgOp int, sliceName string) error {
	logger.ConfigLog.Infof("received slice: %v", sliceName)
	var request configmodels.Slice

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != "application/json" {
		err := fmt.Errorf("unsupported content-type: %s", ct)
		logger.ConfigLog.Errorln(err)
		return err
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ConfigLog.Errorf("JSON bind error: %v", err)
		return err
	}

	logger.ConfigLog.Infof("printing Slice: [%v] received from Roc/Simapp: %+v", sliceName, request)

	for _, gnb := range request.SiteInfo.GNodeBs {
		if !isValidName(gnb.Name) {
			err := fmt.Errorf("invalid gNB name `%s` in Network Slice %s. Name needs to match the following regular expression: %s", gnb.Name, sliceName, NAME_PATTERN)
			logger.ConfigLog.Errorln(err.Error())
			return err
		}
		if !isValidGnbTac(gnb.Tac) {
			err := fmt.Errorf("invalid TAC %d for gNB %s in Network Slice %s. TAC must be an integer within the range [1, 16777215]", gnb.Tac, gnb.Name, sliceName)
			logger.ConfigLog.Errorln(err.Error())
			return err
		}
	}
	slice := request.SliceId
	logger.ConfigLog.Infof("network slice: sst: %v, sd: %v", slice.Sst, slice.Sd)

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)
	logger.ConfigLog.Infof("number of device groups %v", len(request.SiteDeviceGroup))
	for i, g := range request.SiteDeviceGroup {
		logger.ConfigLog.Infof("device groups(%v) - %v", i+1, g)
	}

	for index, filter := range request.ApplicationFilteringRules {
		logger.ConfigLog.Infof("\tRule Name    : %v", filter.RuleName)
		logger.ConfigLog.Infof("\tRule Priority: %v", filter.Priority)
		logger.ConfigLog.Infof("\tRule Action  : %v", filter.Action)
		logger.ConfigLog.Infof("\tEndpoint     : %v", filter.Endpoint)
		logger.ConfigLog.Infof("\tProtocol     : %v", filter.Protocol)
		logger.ConfigLog.Infof("\tStart Port   : %v", filter.StartPort)
		logger.ConfigLog.Infof("\tEnd   Port   : %v", filter.EndPort)
		ul := request.ApplicationFilteringRules[index].AppMbrUplink
		dl := request.ApplicationFilteringRules[index].AppMbrDownlink
		unit := request.ApplicationFilteringRules[index].BitrateUnit

		bitrate := convertToBps(int64(ul), unit)
		if bitrate < 0 || bitrate > math.MaxInt32 {
			request.ApplicationFilteringRules[index].AppMbrUplink = math.MaxInt32
		} else {
			request.ApplicationFilteringRules[index].AppMbrUplink = int32(bitrate)
		}

		bitrate = convertToBps(int64(dl), unit)
		if bitrate < 0 || bitrate > math.MaxInt32 {
			request.ApplicationFilteringRules[index].AppMbrDownlink = math.MaxInt32
		} else {
			request.ApplicationFilteringRules[index].AppMbrDownlink = int32(bitrate)
		}

		logger.ConfigLog.Infof("app MBR Uplink: %v", request.ApplicationFilteringRules[index].AppMbrUplink)
		logger.ConfigLog.Infof("app MBR Downlink: %v", request.ApplicationFilteringRules[index].AppMbrDownlink)
		if filter.TrafficClass != nil {
			logger.ConfigLog.Infof("traffic class: %v", filter.TrafficClass)
		}
	}
	site := request.SiteInfo
	logger.ConfigLog.Infof("site name: %v", site.SiteName)
	logger.ConfigLog.Infof("site PLMN: mcc: %v, mnc: %v", site.Plmn.Mcc, site.Plmn.Mnc)
	logger.ConfigLog.Infof("site gNBs: %v", site.GNodeBs)
	for i, gnb := range site.GNodeBs {
		logger.ConfigLog.Infof("gNB (%v): name=%v, tac=%v", i+1, gnb.Name, gnb.Tac)
	}
	logger.ConfigLog.Infof("site UPF: %v", site.Upf)

	prevSlice := getSliceByName(sliceName)
	if err := handleNetworkSlicePost(&request, &prevSlice); err != nil {
		logger.ConfigLog.Errorf("Error posting slice %v: %v", sliceName, err)
		return err
	}
	var msg configmodels.ConfigMessage
	msg.MsgMethod = msgOp
	request.SliceName = sliceName
	msg.MsgType = configmodels.Network_slice
	msg.Slice = &request
	msg.SliceName = sliceName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Slice [%v] to config channel", sliceName)
	return nil
}
