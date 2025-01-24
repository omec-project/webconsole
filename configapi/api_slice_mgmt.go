// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"math"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/httpwrapper"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
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

func DeviceGroupDeleteHandler(c *gin.Context) bool {
	var groupName string
	var exists bool
	if groupName, exists = c.Params.Get("group-name"); exists {
		logger.ConfigLog.Infof("received Delete Group %v from Roc/simapp", groupName)
	}
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = configmodels.Delete_op
	msg.DevGroupName = groupName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Device Group [%v] with delete_op to config channel", groupName)
	return true
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

func DeviceGroupPostHandler(c *gin.Context, msgOp int) bool {
	groupName, _ := c.Params.Get("group-name")
	if !isValidName(groupName) {
		logger.ConfigLog.Errorf("invalid Device Group name %s. Name needs to match the following regular expression: %s", groupName, NAME_PATTERN)
		return false
	}
	logger.ConfigLog.Infof("received device group: %v", groupName)

	var err error
	var request configmodels.DeviceGroups
	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		logger.ConfigLog.Infof("err %v", err)
		return false
	}
	req := httpwrapper.NewRequest(c.Request, request)

	logger.ConfigLog.Infof("printing Device Group [%v]: %+v", groupName, req)
	logger.ConfigLog.Infof("params: %v", req.Params)
	logger.ConfigLog.Infof("header: %v", req.Header)
	logger.ConfigLog.Infof("query: %v", req.Query)
	logger.ConfigLog.Infof("printing request body: %v", req.Body)
	logger.ConfigLog.Infof("url: %v ", req.URL)

	procReq := req.Body.(configmodels.DeviceGroups)
	ipdomain := &procReq.IpDomainExpanded
	logger.ConfigLog.Infof("imsis.size: %v, Imsis: %v", len(procReq.Imsis), procReq.Imsis)

	logger.ConfigLog.Infof("IP Domain Name: %v", procReq.IpDomainName)
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

	var msg configmodels.ConfigMessage
	procReq.DeviceGroupName = groupName
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = msgOp
	msg.DevGroup = &procReq
	msg.DevGroupName = groupName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully added Device Group [%v] to config channel", groupName)
	return true
}

func NetworkSliceDeleteHandler(c *gin.Context) bool {
	var sliceName string
	var exists bool
	if sliceName, exists = c.Params.Get("slice-name"); exists {
		logger.ConfigLog.Infof("received Deleted slice: %v from Roc/simapp", sliceName)
	}
	var msg configmodels.ConfigMessage
	msg.MsgMethod = configmodels.Delete_op
	msg.MsgType = configmodels.Network_slice
	msg.SliceName = sliceName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Network Slice [%v] with delete_op to config channel", sliceName)
	return true
}

func NetworkSlicePostHandler(c *gin.Context, msgOp int) bool {
	sliceName, _ := c.Params.Get("slice-name")
	if !isValidName(sliceName) {
		logger.ConfigLog.Errorf("invalid Network Slice name %s. Name needs to match the following regular expression: %s", sliceName, NAME_PATTERN)
		return false
	}
	logger.ConfigLog.Infof("received slice: %v", sliceName)

	var err error
	var request configmodels.Slice
	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		logger.ConfigLog.Infof("err %v", err)
		return false
	}

	req := httpwrapper.NewRequest(c.Request, request)

	logger.ConfigLog.Infof("printing Slice: [%v] received from Roc/Simapp: %v", sliceName, request)
	logger.ConfigLog.Infof("params: %v ", req.Params)
	logger.ConfigLog.Infof("header: %v ", req.Header)
	logger.ConfigLog.Infof("query: %v ", req.Query)
	logger.ConfigLog.Infof("printing request body: %v ", req.Body)
	logger.ConfigLog.Infof("url: %v ", req.URL)
	procReq := req.Body.(configmodels.Slice)

	slice := procReq.SliceId
	logger.ConfigLog.Infof("network slice: sst: %v, sd: %v", slice.Sst, slice.Sd)

	group := procReq.SiteDeviceGroup
	slices.Sort(group)
	slices.Compact(group)
	logger.ConfigLog.Infof("number of device groups %v", len(group))
	for i := 0; i < len(group); i++ {
		logger.ConfigLog.Infof("device groups(%v) - %v", i+1, group[i])
	}

	for index, filter := range procReq.ApplicationFilteringRules {
		logger.ConfigLog.Infof("\tRule Name    : %v", filter.RuleName)
		logger.ConfigLog.Infof("\tRule Priority: %v", filter.Priority)
		logger.ConfigLog.Infof("\tRule Action  : %v", filter.Action)
		logger.ConfigLog.Infof("\tEndpoint     : %v", filter.Endpoint)
		logger.ConfigLog.Infof("\tProtocol     : %v", filter.Protocol)
		logger.ConfigLog.Infof("\tStart Port   : %v", filter.StartPort)
		logger.ConfigLog.Infof("\tEnd   Port   : %v", filter.EndPort)
		ul := procReq.ApplicationFilteringRules[index].AppMbrUplink
		dl := procReq.ApplicationFilteringRules[index].AppMbrDownlink
		unit := procReq.ApplicationFilteringRules[index].BitrateUnit

		bitrate := convertToBps(int64(ul), unit)
		if bitrate < 0 || bitrate > math.MaxInt32 {
			procReq.ApplicationFilteringRules[index].AppMbrUplink = math.MaxInt32
		} else {
			procReq.ApplicationFilteringRules[index].AppMbrUplink = int32(bitrate)
		}

		bitrate = convertToBps(int64(dl), unit)
		if bitrate < 0 || bitrate > math.MaxInt32 {
			procReq.ApplicationFilteringRules[index].AppMbrDownlink = math.MaxInt32
		} else {
			procReq.ApplicationFilteringRules[index].AppMbrDownlink = int32(bitrate)
		}

		logger.ConfigLog.Infof("app MBR Uplink: %v", procReq.ApplicationFilteringRules[index].AppMbrUplink)
		logger.ConfigLog.Infof("app MBR Downlink: %v", procReq.ApplicationFilteringRules[index].AppMbrDownlink)
		if filter.TrafficClass != nil {
			logger.ConfigLog.Infof("traffic class: %v", filter.TrafficClass)
		}
	}
	site := procReq.SiteInfo
	logger.ConfigLog.Infof("site name: %v", site.SiteName)
	logger.ConfigLog.Infof("site PLMN: mcc: %v, mnc: %v", site.Plmn.Mcc, site.Plmn.Mnc)
	logger.ConfigLog.Infof("site gNBs: %v", site.GNodeBs)
	for i := 0; i < len(site.GNodeBs); i++ {
		gnb := site.GNodeBs[i]
		logger.ConfigLog.Infof("gNB (%v): name=%v, tac=%v", i+1, gnb.Name, gnb.Tac)
	}
	logger.ConfigLog.Infof("site UPF: %v", site.Upf)

	var msg configmodels.ConfigMessage
	msg.MsgMethod = msgOp
	procReq.SliceName = sliceName
	msg.MsgType = configmodels.Network_slice
	msg.Slice = &procReq
	msg.SliceName = sliceName
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully Added Slice [%v] to config channel", sliceName)
	return true
}
