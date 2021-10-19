// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package configapi

import (
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
	ipdomain := procReq.IpDomainExpanded
	configLog.Infof("Imsis.size : %v, Imsis: %v", len(procReq.Imsis), procReq.Imsis)

	configLog.Infof("IP Domain Name : %v", procReq.IpDomainName)
	configLog.Infof("IP Domain details : %v", ipdomain)
	configLog.Infof("  dnn name : %v", ipdomain.Dnn)
	configLog.Infof("  ue pool  : %v", ipdomain.UeIpPool)
	configLog.Infof("  dns Primary : %v", ipdomain.DnsPrimary)
	configLog.Infof("  dns Secondary : %v", ipdomain.DnsSecondary)
	configLog.Infof("  ip mtu : %v", ipdomain.Mtu)
	configLog.Infof("Device Group Name :  %v ", groupName)

	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Device_group
	msg.MsgMethod = msgOp
	msg.DevGroup = &request
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

	qos := procReq.Qos
	configLog.Infof("Slice QoS   : %v", qos)
	configLog.Infof("  uplink    : %v", qos.Uplink)
	configLog.Infof("  downlink  : %v", qos.Downlink)
	configLog.Infof("  traffic   : %v", qos.TrafficClass)

	group := procReq.SiteDeviceGroup
	configLog.Infof("Number of device groups %v", len(group))
	for i := 0; i < len(group); i++ {
		configLog.Infof("  device groups(%v) - %v \n", i+1, group[i])
	}
	denylist := procReq.DenyApplications
	configLog.Infof("Number of denied applications %v", len(denylist))
	for d := 0; d < len(denylist); d++ {
		configLog.Infof("    deny application %v", denylist[d])
	}
	permitlist := procReq.PermitApplications
	configLog.Infof("Number of permit applications %v", len(permitlist))
	for p := 0; p < len(permitlist); p++ {
		configLog.Infof("    permit application %v", permitlist[p])
	}

	appinfo := procReq.ApplicationsInformation
	configLog.Infof("Length Application information %v", len(appinfo))
	for a := 0; a < len(appinfo); a++ {
		app := appinfo[a]
		configLog.Infof("    appname   : %v", app.AppName)
		configLog.Infof("    endpoint  : %v ", app.Endpoint)
		configLog.Infof("    startPort : %v", app.StartPort)
		configLog.Infof("    endPort   : %v", app.EndPort)
		configLog.Infof("    protocol  : %v", app.Protocol)
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
	msg.MsgType = configmodels.Network_slice
	msg.Slice = &request
	msg.SliceName = sliceName
	configChannel <- &msg
	configLog.Infof("Successfully Added Slice [%v] to config channel.", sliceName)
	return true
}
