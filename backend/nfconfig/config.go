// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	protos "github.com/omec-project/config5g/proto/sdcoreConfig"
	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
)

type accessAndMobilityKey struct {
	plmn    configmodels.SliceSiteInfoPlmn
	sliceId configmodels.SliceSliceId
}

type imsiQosConfig struct {
	imsis []string
	dnn   string
	qos   []nfConfigApi.ImsiQos
}

type inMemoryConfig struct {
	plmn              []nfConfigApi.PlmnId
	plmnSnssai        []nfConfigApi.PlmnSnssai
	accessAndMobility []nfConfigApi.AccessAndMobility
	sessionManagement []nfConfigApi.SessionManagement
	policyControl     []nfConfigApi.PolicyControl
	imsiQos           []imsiQosConfig
}

var defaultPccRule = nfConfigApi.NewPccRule(
	"DefaultRule",
	[]nfConfigApi.PccFlow{
		{
			Description: "permit out ip from any to assigned",
			Direction:   nfConfigApi.DIRECTION_BIDIRECTIONAL,
			Status:      nfConfigApi.STATUS_ENABLED,
		},
	},
	*nfConfigApi.NewPccQos(
		9,
		"1 Kbps",
		"1 Kbps",
		*nfConfigApi.NewArp(
			1,
			nfConfigApi.PREEMPTCAP_MAY_PREEMPT,
			nfConfigApi.PREEMPTVULN_PREEMPTABLE,
		),
	),
	255,
)

func (c *inMemoryConfig) syncPlmn(slices []configmodels.Slice) {
	plmnSet := make(map[nfConfigApi.PlmnId]bool)
	newPlmnConfig := []nfConfigApi.PlmnId{}
	for _, s := range slices {
		plmn := *nfConfigApi.NewPlmnId(s.SiteInfo.Plmn.Mcc, s.SiteInfo.Plmn.Mnc)
		if !plmnSet[plmn] {
			plmnSet[plmn] = true
			newPlmnConfig = append(newPlmnConfig, plmn)
		}
	}

	sort.Slice(newPlmnConfig, func(i, j int) bool {
		if newPlmnConfig[i].GetMcc() != newPlmnConfig[j].GetMcc() {
			return newPlmnConfig[i].GetMcc() < newPlmnConfig[j].GetMcc()
		}
		return newPlmnConfig[i].GetMnc() < newPlmnConfig[j].GetMnc()
	})

	c.plmn = newPlmnConfig
	logger.NfConfigLog.Debugf("Updated PLMN in-memory configuration. New configuration: %+v", c.plmn)
}

func (c *inMemoryConfig) syncPlmnSnssai(slices []configmodels.Slice) {
	plmnMap := make(map[configmodels.SliceSiteInfoPlmn]map[configmodels.SliceSliceId]struct{})
	for _, s := range slices {
		plmn := s.SiteInfo.Plmn
		if plmnMap[plmn] == nil {
			plmnMap[plmn] = map[configmodels.SliceSliceId]struct{}{}
		}
		plmnMap[plmn][s.SliceId] = struct{}{}
	}

	c.plmnSnssai = convertPlmnMapToSortedList(plmnMap)
	logger.NfConfigLog.Debugf("Updated PLMN S-NSSAI in-memory configuration. New configuration: %+v", c.plmnSnssai)
}

func parseSnssaiFromSlice(sliceId configmodels.SliceSliceId) (nfConfigApi.Snssai, error) {
	val, err := strconv.ParseInt(sliceId.Sst, 10, 64)
	if err != nil {
		return *nfConfigApi.NewSnssaiWithDefaults(), err
	}

	snssai := nfConfigApi.NewSnssai(int32(val))
	if sliceId.Sd != "" {
		snssai.SetSd(sliceId.Sd)
	}
	return *snssai, nil
}

func convertPlmnMapToSortedList(plmnMap map[configmodels.SliceSiteInfoPlmn]map[configmodels.SliceSliceId]struct{}) []nfConfigApi.PlmnSnssai {
	newPlmnSnssaiConfig := []nfConfigApi.PlmnSnssai{}
	for plmn, snssaiSet := range plmnMap {
		snssaiList := make([]nfConfigApi.Snssai, 0, len(snssaiSet))
		for snssai := range snssaiSet {
			newSnssai, err := parseSnssaiFromSlice(snssai)
			if err != nil {
				logger.NfConfigLog.Warnf("Error parsing Snssai: %+v. Network slice `%+v` will be ignored", err, snssai)
				continue
			}
			snssaiList = append(snssaiList, newSnssai)
		}
		if len(snssaiList) == 0 {
			continue
		}
		plmnId := nfConfigApi.NewPlmnId(plmn.Mcc, plmn.Mnc)
		plmnSnssai := nfConfigApi.NewPlmnSnssai(*plmnId, snssaiList)
		newPlmnSnssaiConfig = append(newPlmnSnssaiConfig, *plmnSnssai)
	}
	sortPlmnSnssaiConfig(newPlmnSnssaiConfig)
	return newPlmnSnssaiConfig
}

func sortPlmnSnssaiConfig(plmnSnssai []nfConfigApi.PlmnSnssai) {
	sort.Slice(plmnSnssai, func(i, j int) bool {
		if plmnSnssai[i].PlmnId.GetMcc() != plmnSnssai[j].PlmnId.GetMcc() {
			return plmnSnssai[i].PlmnId.GetMcc() < plmnSnssai[j].PlmnId.GetMcc()
		}
		return plmnSnssai[i].PlmnId.GetMnc() < plmnSnssai[j].PlmnId.GetMnc()
	})

	for i := range plmnSnssai {
		sort.Slice(plmnSnssai[i].SNssaiList, func(a, b int) bool {
			s1 := plmnSnssai[i].SNssaiList[a]
			s2 := plmnSnssai[i].SNssaiList[b]
			if s1.GetSst() != s2.GetSst() {
				return s1.GetSst() < s2.GetSst()
			}
			if !s1.HasSd() && s2.HasSd() {
				return true
			}
			if s1.HasSd() && !s2.HasSd() {
				return false
			}
			if s1.HasSd() && s2.HasSd() {
				return s1.GetSd() < s2.GetSd()
			}
			return false
		})
	}
}

func (c *inMemoryConfig) syncAccessAndMobility(networkSlices []configmodels.Slice) {
	plmnSnssaiTacsMap := map[accessAndMobilityKey]map[string]struct{}{}
	for _, s := range networkSlices {
		accessAndMobilityTmp := accessAndMobilityKey{
			plmn:    s.SiteInfo.Plmn,
			sliceId: s.SliceId,
		}
		if plmnSnssaiTacsMap[accessAndMobilityTmp] != nil {
			logger.NfConfigLog.Warnf("Found duplicate Network slice `%+v` for PLMN `%+v`, merging TACs for Access and Mobility", s.SliceId, s.SiteInfo.Plmn)
		} else {
			plmnSnssaiTacsMap[accessAndMobilityTmp] = map[string]struct{}{}
		}
		for _, g := range s.SiteInfo.GNodeBs {
			tac := strconv.Itoa(int(g.Tac))
			plmnSnssaiTacsMap[accessAndMobilityTmp][tac] = struct{}{}
		}
	}
	c.accessAndMobility = convertPlmnSnssaiTacsMapToSortedList(plmnSnssaiTacsMap)
	logger.NfConfigLog.Debugf("Updated Access and Mobility in-memory configuration. New configuration: %+v", c.accessAndMobility)
}

func convertPlmnSnssaiTacsMapToSortedList(plmnSnssaiMap map[accessAndMobilityKey]map[string]struct{}) []nfConfigApi.AccessAndMobility {
	newAccessAndMobilityConfig := []nfConfigApi.AccessAndMobility{}
	for plmnSliceId, tacSet := range plmnSnssaiMap {
		plmnId := nfConfigApi.NewPlmnId(plmnSliceId.plmn.Mcc, plmnSliceId.plmn.Mnc)
		parsedSnssai, err := parseSnssaiFromSlice(plmnSliceId.sliceId)
		if err != nil {
			logger.NfConfigLog.Warnf("Error in parsing SNSSAI: %v. Network slice `%+v` will be ignored", err, plmnSliceId.sliceId)
			continue
		}
		accessAndMobility := nfConfigApi.NewAccessAndMobility(*plmnId, parsedSnssai)
		tacList := make([]string, 0, len(tacSet))
		for tac := range tacSet {
			tacList = append(tacList, tac)
		}
		accessAndMobility.Tacs = tacList
		newAccessAndMobilityConfig = append(newAccessAndMobilityConfig, *accessAndMobility)
	}
	sortAccessAndMobilityConfig(newAccessAndMobilityConfig)
	return newAccessAndMobilityConfig
}

func sortAccessAndMobilityConfig(accessAndMobility []nfConfigApi.AccessAndMobility) {
	sort.Slice(accessAndMobility, func(i, j int) bool {
		if accessAndMobility[i].PlmnId.GetMcc() != accessAndMobility[j].PlmnId.GetMcc() {
			return accessAndMobility[i].PlmnId.GetMcc() < accessAndMobility[j].PlmnId.GetMcc()
		}
		if accessAndMobility[i].PlmnId.GetMnc() != accessAndMobility[j].PlmnId.GetMnc() {
			return accessAndMobility[i].PlmnId.GetMnc() < accessAndMobility[j].PlmnId.GetMnc()
		}
		if accessAndMobility[i].Snssai.GetSst() != accessAndMobility[j].Snssai.GetSst() {
			return accessAndMobility[i].Snssai.GetSst() < accessAndMobility[j].Snssai.GetSst()
		}
		if accessAndMobility[i].Snssai.HasSd() != accessAndMobility[j].Snssai.HasSd() {
			return !accessAndMobility[i].Snssai.HasSd()
		}
		return accessAndMobility[i].Snssai.GetSd() < accessAndMobility[j].Snssai.GetSd()
	})
	for i := range accessAndMobility {
		slices.Sort(accessAndMobility[i].Tacs)
	}
}

func (c *inMemoryConfig) syncSessionManagement(slices []configmodels.Slice, deviceGroupMap map[string]configmodels.DeviceGroups) {
	sessionConfigs := make([]nfConfigApi.SessionManagement, 0, len(slices))

	for _, slice := range slices {
		session, ok := buildSessionManagementConfig(slice, deviceGroupMap)
		if ok {
			sessionConfigs = append(sessionConfigs, *session)
		}
	}

	sort.Slice(sessionConfigs, func(i, j int) bool {
		return sessionConfigs[i].GetSliceName() < sessionConfigs[j].GetSliceName()
	})

	c.sessionManagement = sessionConfigs
	logger.NfConfigLog.Debugf("Updated Session Management configuration with %d slices: %+v", len(sessionConfigs), c.sessionManagement)
}

func buildSessionManagementConfig(slice configmodels.Slice, deviceGroupMap map[string]configmodels.DeviceGroups) (*nfConfigApi.SessionManagement, bool) {
	plmn := nfConfigApi.NewPlmnId(slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc)

	snssai, err := parseSnssaiFromSlice(slice.SliceId)
	if err != nil {
		logger.NfConfigLog.Errorf("Invalid SNSSAI for slice %s: %+v", slice.SliceName, err)
		return nil, false
	}
	session := nfConfigApi.NewSessionManagement(slice.SliceName, *plmn, snssai)

	if ipDomains := extractIpDomains(slice.SiteDeviceGroup, deviceGroupMap); len(ipDomains) > 0 {
		session.SetIpDomain(ipDomains)
	}

	if upf := extractUpf(slice); upf != nil {
		session.SetUpf(*upf)
	}

	if gnbNames := extractGnbNames(slice); len(gnbNames) > 0 {
		session.SetGnbNames(gnbNames)
	}

	return session, true
}

func extractIpDomains(groupNames []string, deviceGroupMap map[string]configmodels.DeviceGroups) []nfConfigApi.IpDomain {
	ipDomains := make([]nfConfigApi.IpDomain, 0, len(groupNames))

	for _, name := range groupNames {
		dg, exists := deviceGroupMap[name]
		if !exists {
			logger.NfConfigLog.Warnf("Device group %s not found", name)
			continue
		}
		ip := nfConfigApi.NewIpDomain(
			dg.IpDomainExpanded.Dnn,
			dg.IpDomainExpanded.DnsPrimary,
			dg.IpDomainExpanded.UeIpPool,
			dg.IpDomainExpanded.Mtu,
		)
		ipDomains = append(ipDomains, *ip)
	}
	return ipDomains
}

func extractUpf(slice configmodels.Slice) *nfConfigApi.Upf {
	upfMap := slice.SiteInfo.Upf
	if upfMap == nil {
		logger.NfConfigLog.Warnf("no UPF defined for slice %s", slice.SliceName)
		return nil
	}
	hostnameRaw, ok := upfMap["upf-name"]
	if !ok {
		logger.NfConfigLog.Warnf("missing UPF hostname for slice %s", slice.SliceName)
		return nil
	}
	hostname, ok := hostnameRaw.(string)
	if !ok || hostname == "" {
		logger.NfConfigLog.Warnf("invalid UPF hostname for slice %s: %v", slice.SliceName, hostnameRaw)
		return nil
	}
	upf := nfConfigApi.NewUpf(hostname)

	if portRaw, ok := upfMap["upf-port"]; ok {
		if portStr, ok := portRaw.(string); ok {
			if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
				upf.SetPort(int32(port))
			} else {
				logger.NfConfigLog.Warnf("invalid UPF port for slice %s: %+v", slice.SliceName, err)
			}
		} else {
			logger.NfConfigLog.Warnf("UPF port should be a string for slice %s, got: %T", slice.SliceName, portRaw)
		}
	}
	return upf
}

func extractGnbNames(slice configmodels.Slice) []string {
	names := make([]string, 0, len(slice.SiteInfo.GNodeBs))
	for _, gnb := range slice.SiteInfo.GNodeBs {
		names = append(names, gnb.Name)
	}
	slices.Sort(names)
	return names
}

func (c *inMemoryConfig) syncPolicyControl(slices []configmodels.Slice) {
	policyControlConfigs := []nfConfigApi.PolicyControl{}

	for _, slice := range slices {
		policyControl, ok := buildPolicyControlConfig(slice)
		if ok {
			policyControlConfigs = append(policyControlConfigs, *policyControl)
		}
	}

	c.policyControl = policyControlConfigs
	logger.NfConfigLog.Debugf("Updated Policy Control in-memory configuration. New configuration: %+v", c.policyControl)
}

func buildPolicyControlConfig(slice configmodels.Slice) (*nfConfigApi.PolicyControl, bool) {
	plmn := nfConfigApi.NewPlmnId(slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc)

	snssai, err := parseSnssaiFromSlice(slice.SliceId)
	if err != nil {
		logger.NfConfigLog.Errorf("Invalid SNSSAI for slice %s: %+v", slice.SliceName, err)
		return nil, false
	}
	pccRules := buildSlicePccRules(slice)
	policyControl := nfConfigApi.NewPolicyControl(*plmn, snssai, pccRules)

	return policyControl, true
}

func buildSlicePccRules(slice configmodels.Slice) []nfConfigApi.PccRule {
	/* Implementation assumes that the validation of a Network Slice configuration is done upon group creation/modification.
	At the time of implementing this, validation is not done, but planned.

	TODO: Remove this comment once Device Group validation is implemented.
	*/
	pccRules := []nfConfigApi.PccRule{}

	for _, ruleConfig := range slice.ApplicationFilteringRules {
		ruleId := ruleConfig.RuleName
		flows := buildPccFlows(ruleConfig)
		qos := buildPccQos(ruleConfig)
		precedence := ruleConfig.Priority

		pccRule := nfConfigApi.NewPccRule(ruleId, flows, qos, precedence)
		pccRules = append(pccRules, *pccRule)
	}

	// If slice has no PCC rules, add a default one
	if len(pccRules) == 0 {
		pccRules = append(pccRules, *defaultPccRule)
	}

	return pccRules
}

func buildPccFlows(ruleConfig configmodels.SliceApplicationFilteringRules) []nfConfigApi.PccFlow {
	pccFlows := []nfConfigApi.PccFlow{}

	description := buildFlowDescription(ruleConfig)
	var status nfConfigApi.Status
	if ruleConfig.Action == "deny" {
		status = nfConfigApi.STATUS_DISABLED
	} else {
		status = nfConfigApi.STATUS_ENABLED
	}

	flowInfo := nfConfigApi.NewPccFlow(
		description,
		nfConfigApi.DIRECTION_BIDIRECTIONAL,
		status,
	)

	pccFlows = append(pccFlows, *flowInfo)
	return pccFlows
}

func buildFlowDescription(ruleConfig configmodels.SliceApplicationFilteringRules) string {
	endp := ruleConfig.Endpoint
	if strings.HasPrefix(endp, "0.0.0.0") {
		endp = "any"
	}

	if ruleConfig.Protocol == int32(protos.PccFlowTos_TCP.Number()) {
		if ruleConfig.StartPort == 0 && ruleConfig.EndPort == 0 {
			return fmt.Sprintf("permit out tcp from %s to assigned", endp)
		} else if factory.WebUIConfig.Configuration.SdfComp {
			return fmt.Sprintf("permit out tcp from %s %s-%s to assigned", endp, strconv.FormatInt(int64(ruleConfig.StartPort), 10), strconv.FormatInt(int64(ruleConfig.EndPort), 10))
		} else {
			return fmt.Sprintf("permit out tcp from %s to assigned %s-%s", endp, strconv.FormatInt(int64(ruleConfig.StartPort), 10), strconv.FormatInt(int64(ruleConfig.EndPort), 10))
		}
	} else if ruleConfig.Protocol == int32(protos.PccFlowTos_UDP.Number()) {
		if ruleConfig.StartPort == 0 && ruleConfig.EndPort == 0 {
			return fmt.Sprintf("permit out udp from %s to assigned", endp)
		} else if factory.WebUIConfig.Configuration.SdfComp {
			return fmt.Sprintf("permit out udp from %s %s-%s to assigned", endp, strconv.FormatInt(int64(ruleConfig.StartPort), 10), strconv.FormatInt(int64(ruleConfig.EndPort), 10))
		} else {
			return fmt.Sprintf("permit out udp from %s to assigned %s-%s", endp, strconv.FormatInt(int64(ruleConfig.StartPort), 10), strconv.FormatInt(int64(ruleConfig.EndPort), 10))
		}
	} else {
		return fmt.Sprintf("permit out ip from %s to assigned", endp)
	}
}

func buildPccQos(ruleConfig configmodels.SliceApplicationFilteringRules) nfConfigApi.PccQos {
	pccQos := nfConfigApi.NewPccQos(
		ruleConfig.TrafficClass.Qci,
		bitrateAsString(ruleConfig.AppMbrUplink),
		bitrateAsString(ruleConfig.AppMbrDownlink),
		*nfConfigApi.NewArp(
			ruleConfig.TrafficClass.Arp,
			nfConfigApi.PREEMPTCAP_MAY_PREEMPT,
			nfConfigApi.PREEMPTVULN_PREEMPTABLE,
		),
	)
	return *pccQos
}

func (c *inMemoryConfig) syncImsiQos(deviceGroupMap map[string]configmodels.DeviceGroups) {
	/* Implementation assumes that the validation of a Device Group configuration is done upon group creation/modification.
	At the time of implementing this, validation is not done, but planned.

	TODO: Remove this comment once Device Group validation is implemented.
	*/
	imsiQosConfigs := []imsiQosConfig{}
	for _, dg := range deviceGroupMap {
		imsiQos := extractQosConfigFromDeviceGroup(dg)
		newImsiQosConfig := imsiQosConfig{
			imsis: dg.Imsis,
			dnn:   dg.IpDomainExpanded.Dnn,
			qos:   []nfConfigApi.ImsiQos{imsiQos},
		}
		imsiQosConfigs = append(imsiQosConfigs, newImsiQosConfig)
	}
	c.imsiQos = imsiQosConfigs
	logger.NfConfigLog.Debugf("Updated IMSI QoS in-memory configuration. New configuration: %+v", c.imsiQos)
}

func extractQosConfigFromDeviceGroup(group configmodels.DeviceGroups) nfConfigApi.ImsiQos {
	return *nfConfigApi.NewImsiQos(
		bitrateAsString(group.IpDomainExpanded.UeDnnQos.DnnMbrUplink),
		bitrateAsString(group.IpDomainExpanded.UeDnnQos.DnnMbrDownlink),
		group.IpDomainExpanded.UeDnnQos.TrafficClass.Qci,
		group.IpDomainExpanded.UeDnnQos.TrafficClass.Arp,
	)
}

type BitrateInt interface {
	~int32 | ~int64
}

func bitrateAsString[T BitrateInt](bitrate T) string {
	const (
		Kbps = 1000
		Mbps = 1000 * Kbps
		Gbps = 1000 * Mbps
	)

	switch {
	case bitrate >= Gbps:
		return fmt.Sprintf("%.2f Gbps", float64(bitrate)/float64(Gbps))
	case bitrate >= Mbps:
		return fmt.Sprintf("%.2f Mbps", float64(bitrate)/float64(Mbps))
	case bitrate >= Kbps:
		return fmt.Sprintf("%.2f Kbps", float64(bitrate)/float64(Kbps))
	default:
		return fmt.Sprintf("%d bps", bitrate)
	}
}
