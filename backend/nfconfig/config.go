// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"slices"
	"sort"
	"strconv"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
)

type accessAndMobilityKey struct {
	plmn    configmodels.SliceSiteInfoPlmn
	sliceId configmodels.SliceSliceId
}

type inMemoryConfig struct {
	plmn              []nfConfigApi.PlmnId
	plmnSnssai        []nfConfigApi.PlmnSnssai
	accessAndMobility []nfConfigApi.AccessAndMobility
	sessionManagement []nfConfigApi.SessionManagement
	policyControl     []nfConfigApi.PolicyControl
}

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
	var sessionConfigs []nfConfigApi.SessionManagement

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
	var ipDomains []nfConfigApi.IpDomain

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
		logger.NfConfigLog.Errorf("no UPF defined for slice %s", slice.SliceName)
		return nil
	}

	// extract UPF hostname
	hostnameRaw, ok := upfMap["upf-name"]
	if !ok {
		logger.NfConfigLog.Errorf("missing UPF hostname for slice %s", slice.SliceName)
		return nil
	}

	hostname, ok := hostnameRaw.(string)
	if !ok || hostname == "" {
		logger.NfConfigLog.Errorf("invalid UPF hostname for slice %s: %v", slice.SliceName, hostnameRaw)
		return nil
	}

	upf := nfConfigApi.NewUpf(hostname)

	// extract UPF port optional
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
	var names []string
	for _, gnb := range slice.SiteInfo.GNodeBs {
		names = append(names, gnb.Name)
	}
	slices.Sort(names)
	return names
}

func (c *inMemoryConfig) syncPolicyControl() {
	c.policyControl = []nfConfigApi.PolicyControl{}
	logger.NfConfigLog.Debugf("Updated Policy Control in-memory configuration. New configuration: %+v", c.policyControl)
}
