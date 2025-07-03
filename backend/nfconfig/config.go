// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"sort"
	"strconv"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
)

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

func (c *inMemoryConfig) syncAccessAndMobility() {
	c.accessAndMobility = []nfConfigApi.AccessAndMobility{}
	logger.NfConfigLog.Debugf("Updated Access and Mobility in-memory configuration. New configuration: %+v", c.accessAndMobility)
}

func (c *inMemoryConfig) syncSessionManagement() {
	c.sessionManagement = []nfConfigApi.SessionManagement{}
	logger.NfConfigLog.Debugf("Updated Session Management in-memory configuration. New configuration: %+v", c.sessionManagement)
}

func (c *inMemoryConfig) syncPolicyControl() {
	c.policyControl = []nfConfigApi.PolicyControl{}
	logger.NfConfigLog.Debugf("Updated Policy Control in-memory configuration. New configuration: %+v", c.policyControl)
}
