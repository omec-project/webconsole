// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"encoding/json"
	"strconv"

	"github.com/omec-project/openapi/nfConfigApi/server"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type inMemoryConfig struct {
	plmn              []server.PlmnId
	plmnSnssai        []server.PlmnSnssai
	accessAndMobility []server.AccessAndMobility
	sessionManagement []server.SessionManagement
	policyControl     []server.PolicyControl
}

func (n *NFConfigServer) syncInMemoryConfig() error {
	if n.inMemoryConfig == nil {
		n.inMemoryConfig = &inMemoryConfig{}
	}

	sliceDataColl := "webconsoleData.snapshots.sliceData"
	rawSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, bson.M{})
	if err != nil {
		return err
	}

	slices := []configmodels.Slice{}
	for _, rawSlice := range rawSlices {
		var s configmodels.Slice
		if err := json.Unmarshal(configmodels.MapToByte(rawSlice), &s); err != nil {
			logger.NfConfigLog.Warnf("Failed to unmarshal slice: %v. Network slice `%s` will be ignored", err, s.SliceName)
			continue
		}
		slices = append(slices, s)
	}

	n.syncPlmnConfig(slices)
	n.syncPlmnSnssaiConfig(slices)
	n.syncAccessAndMobilityConfig()
	n.syncSessionManagementConfig()
	n.syncPolicyControlConfig()
	logger.NfConfigLog.Infoln("Updated NF in-memory configuration")
	return nil
}

func (n *NFConfigServer) syncPlmnConfig(slices []configmodels.Slice) {
	plmnSet := make(map[server.PlmnId]bool)
	result := []server.PlmnId{}
	for _, s := range slices {
		plmn := server.PlmnId{Mcc: s.SiteInfo.Plmn.Mcc, Mnc: s.SiteInfo.Plmn.Mnc}
		if !plmnSet[plmn] {
			plmnSet[plmn] = true
			result = append(result, plmn)
		}
	}

	n.inMemoryConfig.plmn = result
	logger.NfConfigLog.Debugln("Updated PLMN in-memory configuration. New configuration: ", n.inMemoryConfig.plmn)
}

func (n *NFConfigServer) syncPlmnSnssaiConfig(slices []configmodels.Slice) {
	plmnMap := make(map[server.PlmnId]map[server.Snssai]struct{})
	for _, s := range slices {
		plmn := server.PlmnId{Mcc: s.SiteInfo.Plmn.Mcc, Mnc: s.SiteInfo.Plmn.Mnc}
		snssai, err := parseSnssaiFromSlice(s.SliceId)
		if err != nil {
			logger.NfConfigLog.Warnf("Error in parsing SST: %v. Network slice `%s` will be ignored", err, s.SliceName)
			continue
		}

		if plmnMap[plmn] == nil {
			plmnMap[plmn] = map[server.Snssai]struct{}{}
		}
		plmnMap[plmn][snssai] = struct{}{}
	}

	n.inMemoryConfig.plmnSnssai = convertPlmnMapToList(plmnMap)
	logger.NfConfigLog.Debugln("Updated PLMN S-NSSAI in-memory configuration. New configuration: ", n.inMemoryConfig.plmnSnssai)
}

func parseSnssaiFromSlice(sliceId configmodels.SliceSliceId) (server.Snssai, error) {
	val, err := strconv.ParseInt(sliceId.Sst, 10, 64)
	if err != nil {
		return server.Snssai{}, err
	}
	return server.Snssai{Sst: int32(val), Sd: sliceId.Sd}, nil
}

func convertPlmnMapToList(plmnMap map[server.PlmnId]map[server.Snssai]struct{}) []server.PlmnSnssai {
	result := []server.PlmnSnssai{}
	for plmn, snssaiSet := range plmnMap {
		snssaiList := make([]server.Snssai, 0, len(snssaiSet))
		for snssai := range snssaiSet {
			snssaiList = append(snssaiList, snssai)
		}
		result = append(result, server.PlmnSnssai{
			PlmnId:     plmn,
			SNssaiList: snssaiList,
		})
	}
	return result
}

func (n *NFConfigServer) syncAccessAndMobilityConfig() {
	n.inMemoryConfig.accessAndMobility = []server.AccessAndMobility{}
	logger.NfConfigLog.Debugln("Updated Access and Mobility in-memory configuration. New configuration: ", n.inMemoryConfig.accessAndMobility)
}

func (n *NFConfigServer) syncSessionManagementConfig() {
	n.inMemoryConfig.sessionManagement = []server.SessionManagement{}
	logger.NfConfigLog.Debugln("Updated Session Management in-memory configuration. New configuration: ", n.inMemoryConfig.sessionManagement)
}

func (n *NFConfigServer) syncPolicyControlConfig() {
	n.inMemoryConfig.policyControl = []server.PolicyControl{}
	logger.NfConfigLog.Debugln("Updated Policy Control in-memory configuration. New configuration: ", n.inMemoryConfig.policyControl)
}
