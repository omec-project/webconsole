// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type inMemoryConfig struct {
	plmn              []nfConfigApi.PlmnId
	plmnSnssai        []nfConfigApi.PlmnSnssai
	accessAndMobility []nfConfigApi.AccessAndMobility
	sessionManagement []nfConfigApi.SessionManagement
	policyControl     []nfConfigApi.PolicyControl
}

func (n *NFConfigServer) TriggerSync() {
	n.syncMutex.Lock()
	defer n.syncMutex.Unlock()

	if n.syncCancelFunc != nil {
		n.syncCancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.syncCancelFunc = cancel
	logger.NfConfigLog.Debugln("Starting in-memory NF configuration synchronization with new context")
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.NfConfigLog.Infoln("No-op. Sync in-memory configuration was cancelled")
				return
			default:
				err := syncInMemoryConfigFunc(n)
				if err == nil {
					return
				}
				logger.NfConfigLog.Warnf("Sync in-memory configuration failed, retrying: %v", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

var syncInMemoryConfigFunc = func(n *NFConfigServer) error {
	return n.syncInMemoryConfig()
}

func (n *NFConfigServer) syncInMemoryConfig() error {
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
	plmnSet := make(map[nfConfigApi.PlmnId]bool)
	result := []nfConfigApi.PlmnId{}
	for _, s := range slices {
		plmn := nfConfigApi.PlmnId{Mcc: s.SiteInfo.Plmn.Mcc, Mnc: s.SiteInfo.Plmn.Mnc}
		if !plmnSet[plmn] {
			plmnSet[plmn] = true
			result = append(result, plmn)
		}
	}

	n.inMemoryConfig.plmn = result
	logger.NfConfigLog.Debugln("Updated PLMN in-memory configuration. New configuration: ", n.inMemoryConfig.plmn)
}

func (n *NFConfigServer) syncPlmnSnssaiConfig(slices []configmodels.Slice) {
	plmnMap := make(map[configmodels.SliceSiteInfoPlmn]map[configmodels.SliceSliceId]struct{})
	for _, s := range slices {
		plmn := s.SiteInfo.Plmn
		if plmnMap[plmn] == nil {
			plmnMap[plmn] = map[configmodels.SliceSliceId]struct{}{}
		}
		plmnMap[plmn][s.SliceId] = struct{}{}
	}

	n.inMemoryConfig.plmnSnssai = convertPlmnMapToList(plmnMap)
	logger.NfConfigLog.Debugln("Updated PLMN S-NSSAI in-memory configuration. New configuration: ", n.inMemoryConfig.plmnSnssai)
}

func parseSnssaiFromSlice(sliceId configmodels.SliceSliceId) (nfConfigApi.Snssai, error) {
	logger.NfConfigLog.Debugln("Parsing slice ID: ", sliceId)
	val, err := strconv.ParseInt(sliceId.Sst, 10, 64)
	if err != nil {
		return nfConfigApi.Snssai{}, err
	}
	snssai := nfConfigApi.Snssai{
		Sst: int32(val),
	}
	if sliceId.Sd != "" {
		snssai.Sd = &sliceId.Sd
	}
	return snssai, nil
}

func convertPlmnMapToList(plmnMap map[configmodels.SliceSiteInfoPlmn]map[configmodels.SliceSliceId]struct{}) []nfConfigApi.PlmnSnssai {
	result := []nfConfigApi.PlmnSnssai{}
	for plmn, snssaiSet := range plmnMap {
		snssaiList := make([]nfConfigApi.Snssai, 0, len(snssaiSet))
		for snssai := range snssaiSet {
			newSnssai, err := parseSnssaiFromSlice(snssai)
			if err != nil {
				logger.NfConfigLog.Warnf("Error in parsing SST: %v. Network slice `%s` will be ignored", err, snssai)
				continue
			}
			snssaiList = append(snssaiList, newSnssai)
		}
		if len(snssaiList) == 0 {
			continue
		}
		result = append(result, nfConfigApi.PlmnSnssai{
			PlmnId:     nfConfigApi.PlmnId{Mcc: plmn.Mcc, Mnc: plmn.Mnc},
			SNssaiList: snssaiList,
		})
	}
	return result
}

func (n *NFConfigServer) syncAccessAndMobilityConfig() {
	n.inMemoryConfig.accessAndMobility = []nfConfigApi.AccessAndMobility{}
	logger.NfConfigLog.Debugln("Updated Access and Mobility in-memory configuration. New configuration: ", n.inMemoryConfig.accessAndMobility)
}

func (n *NFConfigServer) syncSessionManagementConfig() {
	n.inMemoryConfig.sessionManagement = []nfConfigApi.SessionManagement{}
	logger.NfConfigLog.Debugln("Updated Session Management in-memory configuration. New configuration: ", n.inMemoryConfig.sessionManagement)
}

func (n *NFConfigServer) syncPolicyControlConfig() {
	n.inMemoryConfig.policyControl = []nfConfigApi.PolicyControl{}
	logger.NfConfigLog.Debugln("Updated Policy Control in-memory configuration. New configuration: ", n.inMemoryConfig.policyControl)
}
