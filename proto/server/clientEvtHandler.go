// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package server

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	protos "github.com/omec-project/webconsole/proto/sdcoreConfig"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type clientNF struct {
	id                    string
	rc                    int
	configChanged         bool
	slicesConfigClient    map[string]*configmodels.Slice
	devgroupsConfigClient map[string]*configmodels.DeviceGroups
	outStandingPushConfig chan *configmodels.ConfigMessage
	tempGrpcReq           chan *clientReqMsg
	clientLog             *logrus.Entry
}

//message format received from grpc server thread to Client go routine
type clientReqMsg struct {
	networkSliceReqMsg *protos.NetworkSliceRequest
	grpcRspMsg         chan *clientRspMsg
	newClient          bool
}

//message format to send response from client go routine to grpc server
type clientRspMsg struct {
	networkSliceRspMsg *protos.NetworkSliceResponse
}

var clientNFPool map[string]*clientNF
var restartCounter uint32

func init() {
	clientNFPool = make(map[string]*clientNF)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	restartCounter = r1.Uint32()
}

func getClient(id string) (*clientNF, bool) {
	client := clientNFPool[id]
	if client != nil {
		client.clientLog.Infof("Found client %v ", id)
		return client, false
	}
	logger.GrpcLog.Printf("Created client %v ", id)
	client = &clientNF{}
	subField := logrus.Fields{"NF": id}
	client.clientLog = grpcLog.WithFields(subField)
	client.id = id
	client.outStandingPushConfig = make(chan *configmodels.ConfigMessage, 10)
	client.tempGrpcReq = make(chan *clientReqMsg)
	clientNFPool[id] = client
	client.slicesConfigClient = make(map[string]*configmodels.Slice)
	client.devgroupsConfigClient = make(map[string]*configmodels.DeviceGroups)
	// TODO : should we lock global tables before copying them ?
	for key, value := range slicesConfigSnapshot {
		client.slicesConfigClient[key] = value
	}
	for key, value := range devgroupsConfigSnapshot {
		client.devgroupsConfigClient[key] = value
	}
	go clientEventMachine(client)
	return client, true
}

func fillSite(siteInfoConf *configmodels.SliceSiteInfo, siteInfoProto *protos.SiteInfo) {
	siteInfoProto.SiteName = siteInfoConf.SiteName
	for e := 0; e < len(siteInfoConf.GNodeBs); e++ {
		gnb := siteInfoConf.GNodeBs[e]
		gnbProto := &protos.GNodeB{}
		gnbProto.Name = gnb.Name
		gnbProto.Tac = gnb.Tac
		siteInfoProto.Gnb = append(siteInfoProto.Gnb, gnbProto)
	}
	pl := &protos.PlmnId{}
	pl.Mcc = siteInfoConf.Plmn.Mcc
	pl.Mnc = siteInfoConf.Plmn.Mnc
	siteInfoProto.Plmn = pl

	upf := &protos.UpfInfo{}
	upf.UpfName = siteInfoConf.Upf["upf-name"].(string)
	// TODO panic
	//upf.UpfPort = siteInfoConf.Upf["upf-port"].(uint32)
	siteInfoProto.Upf = upf
}

func fillDeviceGroup(groupName string, devGroupConfig *configmodels.DeviceGroups, devGroupProto *protos.DeviceGroup) {
	devGroupProto.Name = groupName
	ipdomain := &protos.IpDomain{}
	ipdomain.Name = devGroupConfig.IpDomainName
	ipdomain.DnnName = devGroupConfig.IpDomainExpanded.Dnn
	ipdomain.UePool = devGroupConfig.IpDomainExpanded.UeIpPool
	ipdomain.DnsPrimary = devGroupConfig.IpDomainExpanded.DnsPrimary
	ipdomain.Mtu = devGroupConfig.IpDomainExpanded.Mtu
	devGroupProto.IpDomainDetails = ipdomain

	for i := 0; i < len(devGroupConfig.Imsis); i++ {
		devGroupProto.Imsi = append(devGroupProto.Imsi, devGroupConfig.Imsis[i])
	}
}

func fillSlice(client *clientNF, sliceName string, sliceConf *configmodels.Slice, sliceProto *protos.NetworkSlice) bool {
	sliceProto.Name = sliceName
	nssai := &protos.NSSAI{}
	nssai.Sst = sliceConf.SliceId.Sst
	nssai.Sd = sliceConf.SliceId.Sd
	sliceProto.Nssai = nssai
	qos := &protos.QoS{}
	qos.Uplink = sliceConf.Qos.Uplink
	qos.Downlink = sliceConf.Qos.Downlink
	qos.TrafficClass = sliceConf.Qos.TrafficClass
	sliceProto.Qos = qos
	for d := 0; d < len(sliceConf.SiteDeviceGroup); d++ {
		group := sliceConf.SiteDeviceGroup[d]
		client.clientLog.Debugf("group %v, len of devgroupsConfigClient %v ", group, len(client.devgroupsConfigClient))
		devGroupConfig := client.devgroupsConfigClient[group]
		if devGroupConfig == nil {
			client.clientLog.Infoln("Did not find group %v ", group)
			return false
		}
		devGroupProto := &protos.DeviceGroup{}
		fillDeviceGroup(group, devGroupConfig, devGroupProto)
		sliceProto.DeviceGroup = append(sliceProto.DeviceGroup, devGroupProto)
	}
	site := &protos.SiteInfo{}
	sliceProto.Site = site
	fillSite(&sliceConf.SiteInfo, sliceProto.Site)
	// add app info
	for a := 0; a < len(sliceConf.DenyApplications); a++ {
		name := sliceConf.DenyApplications[a]
		sliceProto.DenyApps = append(sliceProto.DenyApps, name)
	}
	for a := 0; a < len(sliceConf.PermitApplications); a++ {
		name := sliceConf.PermitApplications[a]
		sliceProto.PermitApps = append(sliceProto.PermitApps, name)
	}
	//
	//   for a:= 0; a < len(sliceConf.ApplicationsInformation); a++  {
	//
	//   }
	return true
}

func clientEventMachine(client *clientNF) {
	for {
		select {
		case configMsg := <-client.outStandingPushConfig:
			client.clientLog.Infof("Received new configuration for Client %v ", client.id)
			// update config snapshot
			if configMsg.DevGroup != nil {
				client.clientLog.Infof("Received new configuration for device Group  %v ", configMsg.DevGroupName)
				client.devgroupsConfigClient[configMsg.DevGroupName] = configMsg.DevGroup
			} else if configMsg.DevGroupName != "" && configMsg.MsgMethod == configmodels.Delete_op {
				client.clientLog.Infof("Received delete configuration for device Group  %v ", configMsg.DevGroupName)
				client.devgroupsConfigClient[configMsg.DevGroupName] = nil
			}

			if configMsg.Slice != nil {
				client.clientLog.Infof("Received new configuration for slice %v ", configMsg.SliceName)
				client.slicesConfigClient[configMsg.SliceName] = configMsg.Slice
			} else if configMsg.SliceName != "" && configMsg.MsgMethod == configmodels.Delete_op {
				client.clientLog.Infof("Received delete configuration for slice %v ", configMsg.SliceName)
				client.slicesConfigClient[configMsg.SliceName] = nil
			}

			client.configChanged = true

		case cReqMsg := <-client.tempGrpcReq:
			client.clientLog.Infof("Config changed %t and NewClient %t\n", client.configChanged, cReqMsg.newClient)

			sliceDetails := &protos.NetworkSliceResponse{}
			sliceDetails.RestartCounter = restartCounter

			envMsg := &clientRspMsg{}
			envMsg.networkSliceRspMsg = sliceDetails

			if client.configChanged == false && cReqMsg.newClient == false {
				client.clientLog.Infoln("No new update to be sent")
				cReqMsg.grpcRspMsg <- envMsg
				continue
			}
			client.clientLog.Infof("Send complete snapshoot to client. Number of Network Slices %v ", len(client.slicesConfigClient))
			for sliceName, sliceConfig := range client.slicesConfigClient {
				if sliceConfig == nil {
					continue
				}
				sliceProto := &protos.NetworkSlice{}
				result := fillSlice(client, sliceName, sliceConfig, sliceProto)
				if result == true {
					sliceDetails.NetworkSlice = append(sliceDetails.NetworkSlice, sliceProto)
				} else {
					client.clientLog.Infoln("Not sending slice config")
				}
			}
			sliceDetails.ConfigUpdated = 1
			cReqMsg.grpcRspMsg <- envMsg
			client.configChanged = false // TODO RACE CONDITION
		}
	}
}
