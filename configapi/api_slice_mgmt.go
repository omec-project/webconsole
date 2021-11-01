// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package configapi

import (
	"math"
	"strings"

	"github.com/free5gc/http_wrapper"
	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/sirupsen/logrus"
)

var configChannel chan *configmodels.ConfigMessage

var configLog *logrus.Entry

func init() {
	configLog = logger.ConfigLog
}

func SetChannel(cfgChannel chan *configmodels.ConfigMessage) {
	configLog.Infof("Setting configChannel")
	configChannel = cfgChannel
}

func DeviceGroupDeleteHandler(c *gin.Context) bool {
	var groupName string
	var exists bool
	if groupName, exists = c.Params.Get("group-name"); exists {
		configLog.Infof("Received Delete Group %v from Roc/simapp", groupName)
	}
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = configmodels.Delete_op
	msg.DevGroupName = groupName
	configChannel <- &msg
	configLog.Infof("Successfully Added Device Group [%v] with delete_op to config channel.", groupName)
	return true

}

func DeviceGroupPostHandler(c *gin.Context, msgOp int) bool {
	var groupName string
	var exists bool
	if groupName, exists = c.Params.Get("group-name"); exists {
		configLog.Infof("Received group %v", groupName)
	}

	var err error
	var request configmodels.DeviceGroups
	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		configLog.Infof(" err ", err)
		return false
	}
	req := http_wrapper.NewRequest(c.Request, request)

	configLog.Infof("Printing Device Group [%v] : %+v", groupName, req)
	configLog.Infof("params : %v", req.Params)
	configLog.Infof("Header : %v", req.Header)
	configLog.Infof("Query  : %v", req.Query)
	configLog.Infof("Printing request body : %v", req.Body)
	configLog.Infof("URL : %v ", req.URL)

	procReq := req.Body.(configmodels.DeviceGroups)
	ipdomain := &procReq.IpDomainExpanded
	configLog.Infof("Imsis.size : %v, Imsis: %v", len(procReq.Imsis), procReq.Imsis)

	configLog.Infof("IP Domain Name : %v", procReq.IpDomainName)
	configLog.Infof("IP Domain details : %v", ipdomain)
	configLog.Infof("  dnn name : %v", ipdomain.Dnn)
	configLog.Infof("  ue pool  : %v", ipdomain.UeIpPool)
	configLog.Infof("  dns Primary : %v", ipdomain.DnsPrimary)
	configLog.Infof("  dns Secondary : %v", ipdomain.DnsSecondary)
	configLog.Infof("  ip mtu : %v", ipdomain.Mtu)
	configLog.Infof("Device Group Name :  %v ", groupName)
	if ipdomain.UeDnnQos != nil {
		var bitrate int64
		if strings.EqualFold(ipdomain.UeDnnQos.BitrateUnit, "kbps") {
			bitrate = 1000
		} else if strings.EqualFold(ipdomain.UeDnnQos.BitrateUnit, "mbps") {
			bitrate = 1000 * 1000
		} else if strings.EqualFold(ipdomain.UeDnnQos.BitrateUnit, "gbps") {
			bitrate = 1000 * 1000 * 1000
		} else {
			// default consider it as mbps
			bitrate = 1000 * 1000
		}

		ipdomain.UeDnnQos.DnnMbrDownlink = ipdomain.UeDnnQos.DnnMbrDownlink * bitrate
		if ipdomain.UeDnnQos.DnnMbrDownlink < 0 {
			ipdomain.UeDnnQos.DnnMbrDownlink = math.MaxInt64
		}
		ipdomain.UeDnnQos.DnnMbrUplink = ipdomain.UeDnnQos.DnnMbrUplink * bitrate
		if ipdomain.UeDnnQos.DnnMbrUplink < 0 {
			ipdomain.UeDnnQos.DnnMbrUplink = math.MaxInt64
		}
	}

	var msg configmodels.ConfigMessage
	procReq.DeviceGroupName = groupName
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = msgOp
	msg.DevGroup = &procReq
	msg.DevGroupName = groupName
	configChannel <- &msg
	configLog.Infof("Successfully Added Device Group [%v] to config channel.", groupName)
	return true
}

func NetworkSliceDeleteHandler(c *gin.Context) bool {
	var sliceName string
	var exists bool
	if sliceName, exists = c.Params.Get("slice-name"); exists {
		configLog.Infof("Received Deleted slice : %v from Roc/simapp", sliceName)
	}
	var msg configmodels.ConfigMessage
	msg.MsgMethod = configmodels.Delete_op
	msg.MsgType = configmodels.Network_slice
	msg.SliceName = sliceName
	configChannel <- &msg
	configLog.Infof("Successfully Added Device Group [%v] with delete_op to config channel.", sliceName)
	return true
}

func NetworkSlicePostHandler(c *gin.Context, msgOp int) bool {
	var sliceName string
	var exists bool
	if sliceName, exists = c.Params.Get("slice-name"); exists {
		configLog.Infof("Received slice : %v", sliceName)
	}

	var err error
	var request configmodels.Slice
	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		configLog.Infof(" err ", err)
		return false
	}
	//configLog.Infof("Printing request full after binding : %v ", request)

	req := http_wrapper.NewRequest(c.Request, request)

	configLog.Infof("Printing Slice: [%v] received from Roc/Simapp : %v", sliceName, request)
	configLog.Infof("params : %v ", req.Params)
	configLog.Infof("Header : %v ", req.Header)
	configLog.Infof("Query  : %v ", req.Query)
	configLog.Infof("Printing request body : %v ", req.Body)
	configLog.Infof("URL : %v ", req.URL)
	procReq := req.Body.(configmodels.Slice)

	slice := procReq.SliceId
	configLog.Infof("Network Slice : %v", slice)
	configLog.Infof("  sst         : %v", slice.Sst)
	configLog.Infof("  sd          : %v", slice.Sd)

	qos := &procReq.Qos
	var bitrate int64
	if strings.EqualFold(qos.BitrateUnit, "kbps") {
		bitrate = 1000
	} else if strings.EqualFold(qos.BitrateUnit, "mbps") {
		bitrate = 1000 * 1000
	} else if strings.EqualFold(qos.BitrateUnit, "gbps") {
		bitrate = 1000 * 1000 * 1000
	} else {
		// default consider it as mbps
		bitrate = 1000 * 1000
	}
	qos.Uplink = qos.Uplink * bitrate
	if qos.Uplink < 0 {
		qos.Uplink = math.MaxInt32
	}
	qos.Downlink = qos.Downlink * bitrate
	if qos.Downlink < 0 {
		qos.Downlink = math.MaxInt32
	}
	configLog.Infof("Slice QoS ")
	configLog.Infof("  uplink bps    : %v", qos.Uplink)
	configLog.Infof("  downlink bps  : %v", qos.Downlink)
	configLog.Infof("  traffic       : %v", qos.TrafficClass)

	group := procReq.SiteDeviceGroup
	configLog.Infof("Number of device groups %v", len(group))
	for i := 0; i < len(group); i++ {
		configLog.Infof("  device groups(%v) - %v \n", i+1, group[i])
	}
	denylist := procReq.DenyApplications
	if len(denylist) > 0 {
		configLog.Infof("Number of denied applications %v", len(denylist))
		for _, d := range denylist {
			configLog.Infof("    deny application %v", d)
		}
	}
	permitlist := procReq.PermitApplications
	if len(permitlist) > 0 {
		configLog.Infof("Number of permit applications %v", len(permitlist))
		for _, p := range permitlist {
			configLog.Infof("    permit application %v", p)
		}
	}

	appinfo := procReq.ApplicationsInformation
	if len(appinfo) > 0 {
		configLog.Infof("Length Application information %v", len(appinfo))
		for a := 0; a < len(appinfo); a++ {
			app := appinfo[a]
			configLog.Infof("    appname   : %v", app.AppName)
			configLog.Infof("    endpoint  : %v ", app.Endpoint)
			configLog.Infof("    startPort : %v", app.StartPort)
			configLog.Infof("    endPort   : %v", app.EndPort)
			configLog.Infof("    protocol  : %v", app.Protocol)
		}
	}

	for index, filter := range procReq.ApplicationFilteringRules {
		configLog.Infof("\tRule Name        : %v", filter.RuleName)
		configLog.Infof("\tRule Priority    : %v", filter.Priority)
		configLog.Infof("\tRule Action      : %v", filter.Action)
		configLog.Infof("\tEndpoint         : %v", filter.Endpoint)
		configLog.Infof("\tProtocol         : %v", filter.Protocol)
		configLog.Infof("\tStart Port       : %v", filter.StartPort)
		configLog.Infof("\tEnd   Port       : %v", filter.EndPort)
		ul := procReq.ApplicationFilteringRules[index].AppMbrUplink
		dl := procReq.ApplicationFilteringRules[index].AppMbrDownlink
		unit := procReq.ApplicationFilteringRules[index].BitrateUnit
		var bitrate int64
		if strings.EqualFold(unit, "kbps") {
			bitrate = 1000
		} else if strings.EqualFold(unit, "mbps") {
			bitrate = 1000 * 1000
		} else if strings.EqualFold(unit, "gbps") {
			bitrate = 1000 * 1000 * 1000
		} else {
			// default consider it as mbps
			bitrate = 1000 * 1000
		}
		procReq.ApplicationFilteringRules[index].AppMbrUplink = ul * bitrate
		if ul < 0 {
			procReq.ApplicationFilteringRules[index].AppMbrUplink = math.MaxInt64
		}
		procReq.ApplicationFilteringRules[index].AppMbrDownlink = dl * bitrate
		if dl < 0 {
			procReq.ApplicationFilteringRules[index].AppMbrDownlink = math.MaxInt64
		}

		configLog.Infof("\tApp MBR Uplink   : %v", procReq.ApplicationFilteringRules[index].AppMbrUplink)
		configLog.Infof("\tApp MBR Downlink : %v", procReq.ApplicationFilteringRules[index].AppMbrDownlink)
		if filter.TrafficClass != nil {
			configLog.Infof("\t\tTraffic Class : %v", filter.TrafficClass)
		}
	}
	site := procReq.SiteInfo
	configLog.Infof("Site name : %v", site.SiteName)
	configLog.Infof("Site PLMN : %v", site.Plmn)
	configLog.Infof("   mcc    : %v", site.Plmn.Mcc)
	configLog.Infof("   mnc    : %v", site.Plmn.Mnc)
	configLog.Infof("Site gNBs : %v", site.GNodeBs)
	for e := 0; e < len(site.GNodeBs); e++ {
		enb := site.GNodeBs[e]
		configLog.Infof("    enb (%v) - name - %v , tac = %v \n", e+1, enb.Name, enb.Tac)
	}
	configLog.Infof("Site UPF : %v", site.Upf)
	configLog.Infof("    upf-name : %v", site.Upf["upf-name"])
	configLog.Infof("    upf-port : %v", site.Upf["upf-port"])

	var msg configmodels.ConfigMessage
	msg.MsgMethod = msgOp
	procReq.SliceName = sliceName
	msg.MsgType = configmodels.Network_slice
	msg.Slice = &procReq
	msg.SliceName = sliceName
	configChannel <- &msg
	configLog.Infof("Successfully Added Slice [%v] to config channel.", sliceName)
	return true
}
