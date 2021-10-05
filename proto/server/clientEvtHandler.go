// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package server

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	protos "github.com/omec-project/webconsole/proto/sdcoreConfig"
	"github.com/sirupsen/logrus"
)

type clientNF struct {
	id                    string
	rc                    int
	ConfigPushUrl         string
	ConfigCheckUrl        string
	configChanged         bool
	slicesConfigClient    map[string]*configmodels.Slice
	devgroupsConfigClient map[string]*configmodels.DeviceGroups
	outStandingPushConfig chan *configmodels.ConfigMessage
	tempGrpcReq           chan *clientReqMsg
	resStream             protos.ConfigService_NetworkSliceSubscribeServer
	resChannel            chan bool
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

type ServingPlmn struct {
	Mcc int32 `json:"mcc,omitempty"`
	Mnc int32 `json:"mnc,omitempty"`
	Tac int32 `json:"tac,omitempty"`
}

type ImsiRange struct {
	From uint64 `json:"from,omitempty"`
	To   uint64 `json:"to,omitempty"`
}

type selectionKeys struct {
	ServingPlmn  *ServingPlmn `json:"serving-plmn,omitempty"`
	RequestedApn string       `json:"requested-apn,omitempty"`
	ImsiRange    *ImsiRange   `json:"imsi-range,omitempty"`
}

type subSelectionRule struct {
	Keys                     selectionKeys `json:"keys,omitempty"`
	Priority                 int32           `json:"priority,omitempty"`
	SelectedQoSProfile       string        `json:"selected-qos-profile,omitempty"`
	SelectedUserPlaneProfile string        `json:"selected-user-plane-profile,omitempty"`
	SelectedApnProfile       string        `json:"selected-apn-profile,omitempty"`
}

type securityProfile struct {
	Opc string `json:key",omitempty"`
	Key string `json:opc",omitempty"`
	Sqn uint64 `json:sqn",omitempty"`
}

type apnProfile struct {
	DnsPrimary   string `json:"dns_primary,omitempty"`
	DnsSecondary string `json:"dns_secondary,omitempty"`
	ApnName      string `json:"apn-name,omitempty"`
	Mtu          int32  `json:"mtu,omitempty"`
	GxEnabled    bool   `json:"gx_enabled,omitempty"`
}

type userPlaneProfile struct {
	UserPlane     string `json:"user-plane,omitempty"`
	GlobalAddress bool   `json:"global-address,omitempty"`
}

type qosProfile struct {
	Qci  int32     `json:"qci,omitempty"`
	Arp  int32     `json:"arp,omitempty"`
	Ambr []int32 `json:"apn-ambr,omitempty"`
}

type configSpgw struct {
	SubSelectRules    []*subSelectionRule          `json:"subscriber-selection-rules,omitempty"`
	ApnProfiles       map[string]*apnProfile       `json:"apn-profiles,omitempty"`
	UserPlaneProfiles map[string]*userPlaneProfile `json:"user-plane-profiles,omitempty"`
	QosProfiles       map[string]*qosProfile       `json:"qos-profiles,omitempty"`
}

type configHss struct {
	StartImsi   uint64                 `json:"start-imsi,omitempty"`
	EndImsi     uint64                 `json:"end-imsi,omitempty"`
	Opc         string                 `json:"Opc,omitempty"`
	Key         string                 `json:"Key,omitempty"`
	Sqn         uint64                 `json:"sqn,omitempty"`
	Rand        string                 `json:"rand,omitempty"`
	Msisdn      int64                  `json:"msisdn,omitempty"`
	AmbrUl      int32                  `json:"ambr-up,omitempty"`
	AmbrDl      int32                  `json:"ambr-dl,omitempty"`
	ApnProfiles map[string]*apnProfile `json:"apn-profiles,omitempty"`
	Qci         int32                  `json:"qci,omitempty"`
	Arp         int32                  `json:"arp,omitempty"`
}

type ruleFlowInfo struct {
	FlowDesc string `json:"Flow-Description,omitempty"`
	FlowDir  int    `json:"Flow-Direction,omitempty"`
}

type arpInfo struct {
	Priority     int32 `json:"Priority-Level,omitempty"`
	PreEmptCap   int32 `json:"Pre-emption-Capability,omitempty"`
	PreEmpVulner int32 `json:"Pre-emption-Vulnerability,omitempty"`
}

type ruleQosInfo struct {
	Qci       int32    `json:"QoS-Class-Identifier,omitempty"`
	Mbr_ul    int32    `json:"Max-Requested-Bandwidth-UL,omitempty"`
	Mbr_dl    int32    `json:"Max-Requested-Bandwidth-DL,omitempty"`
	Gbr_ul    int32    `json:"Guaranteed-Bitrate-UL,omitempty"`
	Gbr_dl    int32    `json:"Guaranteed-Bitrate-DL,omitempty"`
	Arp       *arpInfo `json:"Allocation-Retention-Priority,omitempty"`
	ApnAmbrUl int32    `json:"APN-Aggregate-Max-Bitrate-UL,omitempty"`
	ApnAmbrDl int32    `json:"APN-Aggregate-Max-Bitrate-DL,omitempty"`
}

type pcrfRuledef struct {
	RuleName   string        `json:"Charging-Rule-Name,omitempty"`
	FlowStatus uint32        `json:"Flow-Status,omitempty"`
	QosInfo    *ruleQosInfo  `json:"QoS-Information,omitempty"`
	FlowInfo   *ruleFlowInfo `json:"Flow-Information,omitempty"`
}

type pcrfRules struct {
	Definitions *pcrfRuledef `json:"definition,omitempty"`
}

type pcrfServices struct {
	Qci                   int32    `json:"qci,omitempty"`
	Arp                   int32    `json:"arp,omitempty"`
	Ambr_ul               int32    `json:"AMBR_UL,omitempty"`
	Ambr_dl               int32    `json:"AMBR_DL,omitempty"`
	Rules                 []string `json:"service-activation-rules,omitempty"`
	Activate_conditions   []string `json:"activate-confitions,omitempty"`
	Deactivate_conditions []string `json:"deactivate-conditions-rules,omitempty"`
	Deactivate_actions    []string `json:"deactivate-actions,omitempty"`
}

type pcrfServiceGroup struct {
	Def_service       []string `json:"default-activate-service,omitempty"`
	OnDemand_services []string `json:"on-demand-service,omitempty"`
}

type PcrfPolicies struct {
	ServiceGroups map[string]*pcrfServiceGroup `json:"service-groups,omitempty"`
	Services      map[string]*pcrfServices     `json:"services,omitempty"`
	Rules         map[string]*pcrfRules        `json:"rules,omitempty"`
}
type configPcrf struct {
	Policies *PcrfPolicies `json:"Policies,omitempty"`
}

type configMme struct {
	PlmnList []string `json:"plmnlist,omitempty"`
}

func init() {
	clientNFPool = make(map[string]*clientNF)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	restartCounter = r1.Uint32()
}

func setClientConfigPushUrl(client *clientNF, url string) {
	client.ConfigPushUrl = url
}

func setClientConfigCheckUrl(client *clientNF, url string) {
	client.ConfigCheckUrl = url
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
	client.tempGrpcReq = make(chan *clientReqMsg, 10)
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
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case t := <-ticker.C:
			if client.ConfigCheckUrl != "" {
				go func() {
					c := &http.Client{}
					httpend := client.ConfigCheckUrl
					req, err := http.NewRequest(http.MethodPost, httpend, nil)
					if err != nil {
						client.clientLog.Infof("An Error Occured %v for channel %v \n", err, t)
					}
					resp, err := c.Do(req)
					if err != nil {
						client.clientLog.Infof("An Error Occured %v\n", err)
					} else {
						if factory.WebUIConfig.Configuration.Mode5G == false && resp.StatusCode == http.StatusNotFound {
							client.clientLog.Infof("Config Check Message POST to %v. Status Code -  %v \n", client.id, resp.StatusCode)
							if client.id == "hss" {
								postConfigHss(client, nil)
							} else if client.id == "mme-app" || client.id == "mme-s1ap" {
								postConfigMme(client)
							} else if client.id == "pcrf" {
								postConfigPcrf(client)
							} else if client.id == "spgw" {
								postConfigSpgw(client)
							}
						}
					}
				}()
			}

		case configMsg := <-client.outStandingPushConfig:
			client.clientLog.Infof("Received new configuration for Client %v ", client.id)
			var lastDevGroup *configmodels.DeviceGroups

			// update config snapshot
			if configMsg.DevGroup != nil {
				lastDevGroup = client.devgroupsConfigClient[configMsg.DevGroupName]
				client.clientLog.Infof("Received new configuration for device Group  %v ", configMsg.DevGroupName)
				client.devgroupsConfigClient[configMsg.DevGroupName] = configMsg.DevGroup
			} else if configMsg.DevGroupName != "" && configMsg.MsgMethod == configmodels.Delete_op {
				lastDevGroup = client.devgroupsConfigClient[configMsg.DevGroupName]
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
			/*If client is attached through stream, then
			  send update to client */
			if client.resStream != nil {
				client.clientLog.Infoln("resStream available")
				var reqMsg clientReqMsg
				var nReq protos.NetworkSliceRequest
				reqMsg.networkSliceReqMsg = &nReq
				reqMsg.grpcRspMsg = make(chan *clientRspMsg)
				reqMsg.newClient = false
				client.tempGrpcReq <- &reqMsg
				client.clientLog.Infoln("sent data to client from push config ")
			}
			if factory.WebUIConfig.Configuration.Mode5G == false {
				//push config to 4G network functions
				if client.id == "hss" {
					if configMsg.MsgType == configmodels.Sub_data && configMsg.MsgMethod == configmodels.Delete_op {
						imsiVal := strings.ReplaceAll(configMsg.Imsi, "imsi-", "")
						deleteConfigHss(client, imsiVal)
					} else {
						postConfigHss(client, lastDevGroup)
					}
				} else if client.id == "mme-app" || client.id == "mme-s1ap" {
					postConfigMme(client)
				} else if client.id == "pcrf" {
					postConfigPcrf(client)
				} else if client.id == "spgw" {
					postConfigSpgw(client)
				}
			}

		case cReqMsg := <-client.tempGrpcReq:
			client.clientLog.Infof("Config changed %t and NewClient %t\n", client.configChanged, cReqMsg.newClient)

			sliceDetails := &protos.NetworkSliceResponse{}
			sliceDetails.RestartCounter = restartCounter

			envMsg := &clientRspMsg{}
			envMsg.networkSliceRspMsg = sliceDetails

			if client.configChanged == false && cReqMsg.newClient == false {
				client.clientLog.Infoln("No new update to be sent")
				if client.resStream == nil {
					cReqMsg.grpcRspMsg <- envMsg
				} else {
					if err := client.resStream.Send(
						envMsg.networkSliceRspMsg); err != nil {
						client.clientLog.Infoln("Failed to send data to client: ", err)
						select {
						case client.resChannel <- true:
							client.clientLog.Infoln("Unsubscribed client: ", client.id)
						default:
							// Default case is to avoid blocking in case client has already unsubscribed
						}
					}
				}
				client.clientLog.Infoln("sent data to client: ")
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
			if client.resStream == nil {
				cReqMsg.grpcRspMsg <- envMsg
			} else {
				if err := client.resStream.Send(
					envMsg.networkSliceRspMsg); err != nil {
					client.clientLog.Infoln("Failed to send data to client: ", err)
					select {
					case client.resChannel <- true:
						client.clientLog.Infoln("Unsubscribed client: ", client.id)
					default:
						// Default case is to avoid blocking in case client has already unsubscribed
					}
				}
			}
			client.clientLog.Infoln("send slice success")
			client.configChanged = false // TODO RACE CONDITION
		}
	}
}

func postConfigMme(client *clientNF) {
	client.clientLog.Infoln("Post configuration to MME")
	config := configMme{}

	for sliceName, sliceConfig := range client.slicesConfigClient {
		if sliceConfig == nil {
			continue
		}
		siteInfo := sliceConfig.SiteInfo
		client.clientLog.Infof("Slice %v, siteInfo.GNodeBs %v", sliceName, siteInfo.GNodeBs)

		//keys.ServingPlmn.Tac = gnb.Tac
		plmn := "mcc=" + siteInfo.Plmn.Mcc + ", mnc=" + siteInfo.Plmn.Mnc

		client.clientLog.Infof("plmn for mme %v", plmn)
		config.PlmnList = append(config.PlmnList, plmn)
	}
	client.clientLog.Infoln("mme Config after filling details ", config)
	b, err := json.Marshal(config)
	if err != nil {
		client.clientLog.Infoln("error in marshalling json -", err)
	} else {
		client.clientLog.Infoln("mme marshalling json -", b)
	}
	reqMsgBody := bytes.NewBuffer(b)
	client.clientLog.Infoln("mme reqMsgBody -", reqMsgBody)
	c := &http.Client{}
	httpend := client.ConfigPushUrl
	req, err := http.NewRequest(http.MethodPost, httpend, reqMsgBody)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.Do(req)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	} else {
		client.clientLog.Infof("mme Message POST %v %v \n", reqMsgBody, resp.StatusCode)
	}
}

func deleteConfigHss(client *clientNF, imsi string) {
	config := configHss{}
	num, _ := strconv.ParseInt(imsi, 10, 64)
	config.StartImsi = uint64(num)
	config.EndImsi = uint64(num)
	client.clientLog.Infoln("HSS config ", config)
	b, err := json.Marshal(config)
	if err != nil {
		client.clientLog.Infoln("error in marshalling json -", err)
	} else {
		client.clientLog.Infoln("marshalling json -", b)
	}
	reqMsgBody := bytes.NewBuffer(b)
	client.clientLog.Infoln("reqMsgBody -", reqMsgBody)
	c := &http.Client{}
	httpend := client.ConfigPushUrl
	req, err := http.NewRequest(http.MethodDelete, httpend, reqMsgBody)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.Do(req)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	} else {
		client.clientLog.Infof("Message DELETE to HSS %v %v Success\n", reqMsgBody, resp.StatusCode)
	}
}

func getDeletedImsiList(prev, curr *configmodels.DeviceGroups) (imsis []string) {
	if curr == nil {
		if prev == nil {
			return
		}
		return prev.Imsis
	}

	for _, pval1 := range prev.Imsis {

		var found bool
		for _, cval2 := range curr.Imsis {
			if pval1 == cval2 {
				found = true
				break
			}
		}
		if !found {
			imsis = append(imsis, pval1)
		}

	}

	return
}

func getAddedImsiList(prev, curr *configmodels.DeviceGroups) (imsis []string) {
	if curr == nil {
		return
	}
	if prev == nil {
		return curr.Imsis
	}

	for _, cval1 := range curr.Imsis {

		var found bool
		for _, pval2 := range prev.Imsis {
			if cval1 == pval2 {
				found = true
				break
			}
		}
		if !found {
			imsis = append(imsis, cval1)
		}

	}

	return
}

func postConfigHss(client *clientNF, lastDevGroup *configmodels.DeviceGroups) {
	client.clientLog.Infoln("Post configuration to Hss")

	for sliceName, sliceConfig := range client.slicesConfigClient {
		if sliceConfig == nil {
			continue
		}
		client.clientLog.Infoln("SliceName ", sliceName)

		for _, d := range sliceConfig.SiteDeviceGroup {
			if devgroupsConfigSnapshot[d] == nil {
				client.clientLog.Errorln("Device Group is deleted: ", d)
				imsis := getDeletedImsiList(lastDevGroup, nil)
				for _, val := range imsis {
					deleteConfigHss(client, val)
				}
				continue
			}
			config := configHss{
				ApnProfiles: make(map[string]*apnProfile),
			}
			devGroup := devgroupsConfigSnapshot[d]
			// qos profile
			sqos := sliceConfig.Qos
			config.Qci, config.Arp = parseTrafficClass(devGroup.IpDomainExpanded.ApnQos.TrafficClass)
			config.AmbrUl = sqos.Uplink
			config.AmbrDl = sqos.Downlink
			client.clientLog.Infoln("DeviceGroup ", devGroup)
			var apnProf apnProfile
			apnProf.ApnName = devGroup.IpDomainExpanded.Dnn
			apnProfName := sliceName + "-apn"
			config.ApnProfiles[apnProfName] = &apnProf

			if lastDevGroup != nil {
				// imsi is not present in latest device Group
				imsis := getDeletedImsiList(lastDevGroup, devGroup)
				client.clientLog.Infoln("Deleted Imsi list from DeviceGroup: ", imsis)
				for _, val := range imsis {
					deleteConfigHss(client, val)
				}
			}

			newImsis := getAddedImsiList(lastDevGroup, devGroup)

			for _, imsi := range newImsis {
				num, _ := strconv.ParseInt(imsi, 10, 64)
				config.StartImsi = uint64(num)
				config.EndImsi = uint64(num)
				authSubsData := imsiData[imsi]
				client.clientLog.Infoln("imsiData ", imsiData)
				if authSubsData == nil {
					client.clientLog.Infoln("SIM card details not found for IMSI ", imsi)
					continue
				}
				config.Opc = authSubsData.Opc.OpcValue
				config.Key = authSubsData.PermanentKey.PermanentKeyValue
				num, _ = strconv.ParseInt(authSubsData.SequenceNumber, 10, 64)
				config.Sqn = uint64(num)
				client.clientLog.Infoln("HSS config ", config)
				b, err := json.Marshal(config)
				if err != nil {
					client.clientLog.Infoln("error in marshalling json -", err)
				} else {
					client.clientLog.Infoln("marshalling json -", b)
				}
				reqMsgBody := bytes.NewBuffer(b)
				client.clientLog.Infoln("reqMsgBody -", reqMsgBody)
				c := &http.Client{}
				httpend := client.ConfigPushUrl
				req, err := http.NewRequest(http.MethodPost, httpend, reqMsgBody)
				if err != nil {
					client.clientLog.Infof("An Error Occured %v", err)
				}
				req.Header.Set("Content-Type", "application/json; charset=utf-8")
				resp, err := c.Do(req)
				if err != nil {
					client.clientLog.Infof("An Error Occured %v", err)
				} else {
					client.clientLog.Infof("Message POST to HSS %v %v Success\n", reqMsgBody, resp.StatusCode)
				}

			}
			// multiple groups handling?
		}
	}
}

func parseTrafficClass(traffic string) (int32, int32) {
    switch traffic {
        case "silver":
            return 9, 0x7D
        case "platinum":
            return 8, 0x7D
        case "gold":
            return 7, 0x7D
        case "diamond":
            return 6, 0x7D
        default:
            return 9, 0x7D
    }
}

func postConfigPcrf(client *clientNF) {
	client.clientLog.Infoln("Post configuration to Pcrf")
	config := configPcrf{}
	config.Policies = &PcrfPolicies{
		ServiceGroups: make(map[string]*pcrfServiceGroup),
		Services:      make(map[string]*pcrfServices),
		Rules:         make(map[string]*pcrfRules),
	}

	for sliceName, sliceConfig := range client.slicesConfigClient {
		if sliceConfig == nil {
			continue
		}
		client.clientLog.Infoln("Slice ", sliceName)
		siteInfo := sliceConfig.SiteInfo
		client.clientLog.Infoln("siteInfo ", siteInfo)
		//subscriber selection rules
		rule := subSelectionRule{}
		rule.Priority = 1
		//apn profile
		for _, d := range sliceConfig.SiteDeviceGroup {
			devGroup := devgroupsConfigSnapshot[d]
			if devGroup == nil {
				client.clientLog.Errorln("Device Group doesn't exist: ", d)
				continue
			}
			client.clientLog.Infoln("PCRF devgroup ", d)
			sgroup := &pcrfServiceGroup{}
			pcrfServiceName := d + "-service"
			sgroup.Def_service = append(sgroup.Def_service, pcrfServiceName)
			config.Policies.ServiceGroups[devGroup.IpDomainExpanded.Dnn] = sgroup
			pcrfService := &pcrfServices{}
			pcrfService.Qci, pcrfService.Arp = parseTrafficClass(devGroup.IpDomainExpanded.ApnQos.TrafficClass) /* map traffic class to QCI, ARP */
			pcrfService.Ambr_ul = devGroup.IpDomainExpanded.ApnQos.Uplink
			pcrfService.Ambr_dl = devGroup.IpDomainExpanded.ApnQos.Downlink
			if len(sliceConfig.ApplicationFilteringRules) == 0 {
				app := configmodels.SliceApplicationFilteringRules{RuleName: "rule1", Priority: 1, Action: "permit", Endpoint: "0.0.0.0/0"}
				sliceConfig.ApplicationFilteringRules = append(sliceConfig.ApplicationFilteringRules, app)
			}
			for _, app := range sliceConfig.ApplicationFilteringRules {
				ruleName := d + app.RuleName
				client.clientLog.Infoln("rulename ", ruleName)
				pcrfService.Rules = append(pcrfService.Rules, ruleName)
				client.clientLog.Infoln("pcrf Service ", pcrfService.Rules)
				config.Policies.Services[pcrfServiceName] = pcrfService
				pcrfRule := &pcrfRules{}
				ruledef := &pcrfRuledef{}
				pcrfRule.Definitions = ruledef
				ruledef.RuleName = ruleName
				ruledef.FlowStatus = 3 // disabled by default
                if app.Action == "permit" {
				  ruledef.FlowStatus = 2
                }
				ruleQInfo := &ruleQosInfo{}
				ruledef.QosInfo = ruleQInfo
                var arpi int32
				ruleQInfo.Qci, arpi = parseTrafficClass(app.TrafficClass)
				ruleQInfo.Mbr_ul = app.AppMbrUplink
				ruleQInfo.Mbr_dl = app.AppMbrDownlink
				ruleQInfo.Gbr_ul = 0
				ruleQInfo.Gbr_dl = 0
				ruleQInfo.ApnAmbrUl = devGroup.IpDomainExpanded.ApnQos.Uplink
				ruleQInfo.ApnAmbrDl = devGroup.IpDomainExpanded.ApnQos.Downlink
				arp := &arpInfo{}
				arp.Priority = (arpi & 0x3c) >> 2
				arp.PreEmptCap = (arpi & 0x40) >> 6
				arp.PreEmpVulner = arpi & 0x1
				ruleQInfo.Arp = arp
				ruleFInfo := &ruleFlowInfo{}
				// permit out udp from 8.8.8.8/32 to assigned sport-dport
				var desc string
				if app.Protocol == 6 {
					desc = "permit out tcp from " + app.Endpoint + " to assigned " + strconv.FormatInt(int64(app.StartPort), 10) + "-" + strconv.FormatInt(int64(app.EndPort), 10)
				} else if app.Protocol == 17 {
					desc = "permit out udp from " + app.Endpoint + " to assigned " + strconv.FormatInt(int64(app.StartPort), 10) + "-" + strconv.FormatInt(int64(app.EndPort), 10)
				} else {
					desc = "permit out ip from " + app.Endpoint + " to assigned"
				}
				ruleFInfo.FlowDesc = desc
				ruleFInfo.FlowDir = 3
				ruledef.FlowInfo = ruleFInfo
				config.Policies.Rules[ruleName] = pcrfRule
			}
		}
	}

	client.clientLog.Infoln("PCRF Config after filling details ", config)
	client.clientLog.Infoln("PCRF Config after filling details ", config.Policies)
	b, err := json.Marshal(config)
	if err != nil {
		client.clientLog.Infoln("PCRF error in marshalling json -", err)
	} else {
		client.clientLog.Infoln("PCRF marshalling json -", b)
	}
	reqMsgBody := bytes.NewBuffer(b)
	client.clientLog.Infoln("PCRF reqMsgBody -", reqMsgBody)
	c := &http.Client{}
	httpend := client.ConfigPushUrl
	req, err := http.NewRequest(http.MethodPost, httpend, reqMsgBody)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.Do(req)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	} else {
		client.clientLog.Infof("PCRF Message POST %v %v Success\n", reqMsgBody, resp.StatusCode)
	}
}

func postConfigSpgw(client *clientNF) {
	client.clientLog.Infoln("Post configuration to spgw ", client.slicesConfigClient)
	config := configSpgw{
		ApnProfiles:       make(map[string]*apnProfile),
		UserPlaneProfiles: make(map[string]*userPlaneProfile),
		QosProfiles:       make(map[string]*qosProfile),
	}

	for sliceName, sliceConfig := range client.slicesConfigClient {
		if sliceConfig == nil {
			continue
		}
		siteInfo := sliceConfig.SiteInfo
		client.clientLog.Infoln("siteInfo.GNodeBs ", siteInfo.GNodeBs)
		for _, d := range sliceConfig.SiteDeviceGroup {
			devGroup := devgroupsConfigSnapshot[d]
			if devGroup == nil {
				client.clientLog.Errorln("Device Group is not exist: ", d)
				continue
			}
			var rule subSelectionRule
			rule.Priority = 1
			var apnProf apnProfile
			apnProf.DnsPrimary = devGroup.IpDomainExpanded.DnsPrimary
			apnProf.DnsSecondary = devGroup.IpDomainExpanded.DnsPrimary
			apnProf.ApnName = devGroup.IpDomainExpanded.Dnn
			apnProf.Mtu = devGroup.IpDomainExpanded.Mtu
			apnProf.GxEnabled = false
			apnProfName := sliceName + "-apn"
			config.ApnProfiles[apnProfName] = &apnProf
			rule.SelectedApnProfile = apnProfName

			// user plane profile
			var upProf userPlaneProfile
			userProfName := sliceName + "_up"
			upProf.UserPlane = siteInfo.Upf["upf-name"].(string)
			upProf.GlobalAddress = true
			config.UserPlaneProfiles[userProfName] = &upProf
			rule.SelectedUserPlaneProfile = userProfName

			// qos profile
			sqos := sliceConfig.Qos
			qosProfName := sliceName + "_qos"
			var qosProf qosProfile
			qosProf.Qci = 9
			qosProf.Arp = 1
			qosProf.Ambr = append(qosProf.Ambr, sqos.Uplink)
			qosProf.Ambr = append(qosProf.Ambr, sqos.Downlink)
			config.QosProfiles[qosProfName] = &qosProf
			rule.SelectedQoSProfile = qosProfName

			var key selectionKeys
			key.RequestedApn = devGroup.IpDomainExpanded.Dnn
			rule.Keys = key
			config.SubSelectRules = append(config.SubSelectRules, &rule)
		}
	}
	client.clientLog.Infoln("spgw Config after filling details ", config)
	b, err := json.Marshal(config)
	if err != nil {
		client.clientLog.Infoln("error in marshalling json -", err)
	} else {
		client.clientLog.Infoln("spgw marshalling json -", b)
	}
	reqMsgBody := bytes.NewBuffer(b)
	client.clientLog.Infoln("spgw reqMsgBody -", reqMsgBody)
	c := &http.Client{}
	httpend := client.ConfigPushUrl
	req, err := http.NewRequest(http.MethodPost, httpend, reqMsgBody)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.Do(req)
	if err != nil {
		client.clientLog.Infof("An Error Occured %v", err)
	} else {
		client.clientLog.Infof("spgw Message POST %v %v Success\n", reqMsgBody, resp.StatusCode)
	}
}
