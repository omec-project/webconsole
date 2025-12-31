// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"
)

var (
	SyncSliceStop      bool = false
	syncSliceStopMutex sync.Mutex
)

var execCommand = exec.Command

func networkSliceDeleteHelper(sliceName string) error {
	if err := handleNetworkSliceDelete(sliceName); err != nil {
		logger.ConfigLog.Errorf("Error deleting slice %s: %+v", sliceName, err)
		return err
	}
	return nil
}

func networkSlicePostHelper(c *gin.Context, sliceName string) (int, error) {
	logger.ConfigLog.Infof("received slice: %s", sliceName)
	requestSlice, err := parseAndValidateSliceRequest(c, sliceName)
	if err != nil {
		return http.StatusBadRequest, err
	}

	logSliceMetadata(requestSlice)
	normalizeApplicationFilteringRules(&requestSlice)
	requestSlice.SliceName = sliceName
	prevSlice := getSliceByName(sliceName)

	if prevSlice == nil {
		logger.ConfigLog.Infof("Adding new slice [%s]", sliceName)
		if statusCode, err := createNS(requestSlice); err != nil {
			logger.ConfigLog.Errorf("Error creating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	} else {
		if statusCode, err := updateNS(requestSlice, *prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice %s: %+v", sliceName, err)
			return statusCode, err
		}
	}
	return http.StatusOK, nil
}

func parseAndValidateSliceRequest(c *gin.Context, sliceName string) (configmodels.Slice, error) {
	var request configmodels.Slice

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != "application/json" {
		return request, fmt.Errorf("unsupported content-type: %s", ct)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		return request, fmt.Errorf("JSON bind error: %+v", err)
	}

	for i, gnb := range request.SiteInfo.GNodeBs {
		if !isValidName(gnb.Name) {
			return request, fmt.Errorf("invalid gNodeBs[%d].name `%s` in Network Slice %s", i, gnb.Name, sliceName)
		}
		if !isValidGnbTac(gnb.Tac) {
			return request, fmt.Errorf("invalid gNodeBs[%d].tac %d for gNB %s in Network Slice %s", i, gnb.Tac, gnb.Name, sliceName)
		}
	}

	request.SliceName = sliceName
	// Validate required fields are not empty
	if strings.TrimSpace(request.SliceName) == "" {
		return request, fmt.Errorf("slice-name cannot be empty")
	}
	if strings.TrimSpace(request.SliceId.Sst) == "" {
		return request, fmt.Errorf("slice-id.sst cannot be empty")
	}
	if strings.TrimSpace(request.SliceId.Sd) == "" {
		return request, fmt.Errorf("slice-id.sd cannot be empty")
	}
	if len(request.SiteDeviceGroup) == 0 {
		return request, fmt.Errorf("site-device-group cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.SiteName) == "" {
		return request, fmt.Errorf("site-info.site-name cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.Plmn.Mcc) == "" {
		return request, fmt.Errorf("site-info.plmn.mcc cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.Plmn.Mnc) == "" {
		return request, fmt.Errorf("site-info.plmn.mnc cannot be empty")
	}
	if request.SiteInfo.Upf == nil {
		return request, fmt.Errorf("site-info.upf cannot be empty")
	}
	if len(request.SiteInfo.GNodeBs) == 0 {
		return request, fmt.Errorf("site-info.gNodeBs cannot be empty")
	}
	for i, gnodeb := range request.SiteInfo.GNodeBs {
		if strings.TrimSpace(gnodeb.Name) == "" {
			return request, fmt.Errorf("site-info.gNodeBs[%d].name cannot be empty", i)
		}
		if gnodeb.Tac <= 0 {
			return request, fmt.Errorf("site-info.gNodeBs[%d].tac must be > 0", i)
		}
	}

	// Validate ApplicationFilteringRules
	// Si no hay reglas de filtrado, agrega una por defecto
	if len(request.ApplicationFilteringRules) == 0 {
		request.ApplicationFilteringRules = append(request.ApplicationFilteringRules, configmodels.SliceApplicationFilteringRules{
			RuleName:       "default",
			Action:         "permit",
			Endpoint:       "any",
			Protocol:       0,
			StartPort:      0,
			EndPort:        65535,
			AppMbrUplink:   0,
			AppMbrDownlink: 0,
			BitrateUnit:    "bps",
			TrafficClass: &configmodels.TrafficClassInfo{
				Name: "default",
				Qci:  9,
				Arp:  8,
				Pdb:  100,
				Pelr: 6,
			},
		})
	} else {
		for i, rule := range request.ApplicationFilteringRules {
			if strings.TrimSpace(rule.RuleName) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: rule-name cannot be empty", i)
			}
			if strings.TrimSpace(rule.Action) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: action cannot be empty", i)
			}
			if strings.TrimSpace(rule.Endpoint) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: endpoint cannot be empty", i)
			}
			if rule.Protocol < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: protocol must be >= 0", i)
			}
			if rule.StartPort < 0 || rule.EndPort < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: port values must be >= 0", i)
			}
			if rule.EndPort < rule.StartPort {
				return request, fmt.Errorf("application-filtering-rules[%d]: dest-port-end must be >= dest-port-start", i)
			}
			if rule.AppMbrUplink < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: app-mbr-uplink must be >= 0", i)
			}
			if rule.AppMbrDownlink < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: app-mbr-downlink must be >= 0", i)
			}
			if rule.BitrateUnit == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: bitrate-unit cannot be empty", i)
			}
			if rule.TrafficClass != nil {
				if strings.TrimSpace(rule.TrafficClass.Name) == "" {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.name cannot be empty", i)
				}
				if rule.TrafficClass.Qci < 1 || rule.TrafficClass.Qci > 9 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.qci must be between 1 and 9", i)
				}
				if rule.TrafficClass.Arp < 1 || rule.TrafficClass.Arp > 15 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.arp must be between 1 and 15", i)
				}
				if rule.TrafficClass.Pdb < 0 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.pdb must be >= 0", i)
				}
				if rule.TrafficClass.Pelr < 1 || rule.TrafficClass.Pelr > 8 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.pelr must be between 1 and 8", i)
				}
			}
			if rule.TrafficClass == nil {
				return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class cannot be empty", i)
			}
		}
	}

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)

	return request, nil
}

func logSliceMetadata(slice configmodels.Slice) {
	logger.ConfigLog.Infof("network slice: sst: %s, sd: %s", slice.SliceId.Sst, slice.SliceId.Sd)
	logger.ConfigLog.Infof("number of device groups %v", len(slice.SiteDeviceGroup))
	for i, g := range slice.SiteDeviceGroup {
		logger.ConfigLog.Infof("device groups(%d) - %s", i+1, g)
	}

	site := slice.SiteInfo
	logger.ConfigLog.Infof("site name: %s", site.SiteName)
	logger.ConfigLog.Infof("site PLMN: mcc: %s, mnc: %s", site.Plmn.Mcc, site.Plmn.Mnc)
	for i, gnb := range site.GNodeBs {
		logger.ConfigLog.Infof("gNB (%d): name=%s, tac=%d", i+1, gnb.Name, gnb.Tac)
	}
	logger.ConfigLog.Infof("site UPF: %s", site.Upf)
}

func normalizeApplicationFilteringRules(slice *configmodels.Slice) {
	for i := range slice.ApplicationFilteringRules {
		rule := &slice.ApplicationFilteringRules[i]
		logger.ConfigLog.Infof("Rule [%d] Name: %s, Action: %s, Endpoint: %s", i, rule.RuleName, rule.Action, rule.Endpoint)

		ul := convertToBps(int64(rule.AppMbrUplink), rule.BitrateUnit)
		rule.AppMbrUplink = convertBitrateToInt32(ul)

		dl := convertToBps(int64(rule.AppMbrDownlink), rule.BitrateUnit)
		rule.AppMbrDownlink = convertBitrateToInt32(dl)

		logger.ConfigLog.Infof("Normalized MBR Uplink: %v, Downlink: %v", rule.AppMbrUplink, rule.AppMbrDownlink)
		if rule.TrafficClass != nil {
			logger.ConfigLog.Infof("Traffic class: %v", rule.TrafficClass)
		}
	}
}

func convertBitrateToInt32(bitrate int64) int32 {
	if bitrate < 0 || bitrate > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(bitrate)
}

func createNS(slice configmodels.Slice) (int, error) {
	if statusCode, err := handleNetworkSlicePost(slice, configmodels.Slice{}); err != nil {
		logger.ConfigLog.Errorf("Error creating slice %s: %+v", slice.SliceName, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func updateNS(slice, prevSlice configmodels.Slice) (int, error) {
	if statusCode, err := handleNetworkSlicePost(slice, prevSlice); err != nil {
		logger.ConfigLog.Errorf("Error updating slice %s: %+v", slice.SliceName, err)
		return statusCode, err
	}
	return http.StatusOK, nil
}

func handleNetworkSlicePost(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
	filter := bson.M{"slice-name": slice.SliceName}
	sliceDataBsonA := configmodels.ToBsonM(slice)
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(sliceDataColl, filter, sliceDataBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to post slice data for %s: %+v", slice.SliceName, err)
		return http.StatusInternalServerError, err
	}
	logger.AppLog.Debugf("succeeded to post slice data for %s", slice.SliceName)

	statusCode, err := syncSubConcurrently(slice, prevSlice)
	if err != nil {
		return statusCode, err
	}
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err = sendPebbleNotification("aetherproject.org/webconsole/networkslice/create")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return http.StatusOK, nil
}

func syncSubConcurrently(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
	syncSliceStopMutex.Lock()
	if SyncSliceStop {
		syncSliceStopMutex.Unlock()
		return http.StatusServiceUnavailable, errors.New("error: the sync function is running")
	}
	SyncSliceStop = true
	syncSliceStopMutex.Unlock()

	go func() {
		defer func() {
			syncSliceStopMutex.Lock()
			SyncSliceStop = false
			syncSliceStopMutex.Unlock()
		}()
		_, err := syncSubscribersOnSliceCreateOrUpdate(slice, prevSlice)
		if err != nil {
			logger.AppLog.Errorf("error syncing subscribers: %s", err)
		}
	}()

	return 0, nil
}

func sendPebbleNotification(key string) error {
	cmd := execCommand("pebble", "notify", key)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't execute a pebble notify: %+v", err)
	}
	logger.ConfigLog.Infoln("custom Pebble notification sent")
	return nil
}

var syncSubscribersOnSliceDelete = func(slice *configmodels.Slice, prevSlice *configmodels.Slice) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	if slice == nil && prevSlice != nil {
		logger.WebUILog.Debugf("Deleted slice: %s", prevSlice.SliceName)
		return cleanupDeviceGroups(configmodels.Slice{}, *prevSlice)
	}
	return nil
}

var syncSubscribersOnSliceCreateOrUpdate = func(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
	rwLock.Lock()
	defer rwLock.Unlock()
	logger.WebUILog.Debugln("insert/update Slice:", slice)
	logger.AppLog.Debugf("syncSubscribersOnSliceCreateOrUpdate: slice=%s deviceGroups=%d", slice.SliceName, len(slice.SiteDeviceGroup))
	if slice.SliceId.Sst == "" {
		err := fmt.Errorf("missing SST in slice %s", slice.SliceName)
		logger.AppLog.Error(err)
		return http.StatusBadRequest, err
	}
	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
	if err != nil {
		logger.AppLog.Errorf("could not parse SST %s", slice.SliceId.Sst)
		return http.StatusBadRequest, err
	}
	snssai := &models.Snssai{
		Sd:  slice.SliceId.Sd,
		Sst: int32(sVal),
	}
	for _, dgName := range slice.SiteDeviceGroup {
		logger.ConfigLog.Debugf("dgName: %s", dgName)
		devGroupConfig := getDeviceGroupByName(dgName)
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found: %s", dgName)
			continue
		}
		logger.AppLog.Debugf("slice=%s dg=%s: inputIMSIs=%d", slice.SliceName, dgName, len(devGroupConfig.Imsis))

		existing, err := filterExistingIMSIsFromAuthDB(devGroupConfig.Imsis)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		if len(existing) == 0 {
			logger.AppLog.Debugf("slice=%s dg=%s: no existing IMSIs after auth filter", slice.SliceName, dgName)
			continue
		}
		logger.AppLog.Debugf("slice=%s dg=%s: existingIMSIs=%d", slice.SliceName, dgName, len(existing))

		if err := updatePolicyAndProvisionedDataBatch(
			existing,
			slice.SiteInfo.Plmn.Mcc,
			slice.SiteInfo.Plmn.Mnc,
			snssai,
			devGroupConfig.IpDomainExpanded.Dnn,
			devGroupConfig.IpDomainExpanded.UeDnnQos,
		); err != nil {
			logger.AppLog.Errorf("batch update failed for device group %s: %v", dgName, err)
			return http.StatusInternalServerError, err
		}
		logger.AppLog.Debugf("slice=%s dg=%s: batch updates complete", slice.SliceName, dgName)
	}
	if err := cleanupDeviceGroups(slice, prevSlice); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// var syncSubscribersOnSliceCreateOrUpdatev2 = func(slice configmodels.Slice, prevSlice configmodels.Slice) (int, error) {
// 	rwLock.Lock()
// 	defer rwLock.Unlock()
// 	logger.WebUILog.Debugln("insert/update Slice:", slice)
// 	logger.AppLog.Debugf("syncSubscribersOnSliceCreateOrUpdate: slice=%s deviceGroups=%d", slice.SliceName, len(slice.SiteDeviceGroup))
// 	if slice.SliceId.Sst == "" {
// 		err := fmt.Errorf("missing SST in slice %s", slice.SliceName)
// 		logger.AppLog.Error(err)
// 		return http.StatusBadRequest, err
// 	}
// 	sVal, err := strconv.ParseUint(slice.SliceId.Sst, 10, 32)
// 	if err != nil {
// 		logger.AppLog.Errorf("could not parse SST %s", slice.SliceId.Sst)
// 		return http.StatusBadRequest, err
// 	}
// 	snssai := &models.Snssai{
// 		Sd:  slice.SliceId.Sd,
// 		Sst: int32(sVal),
// 	}
// 	for _, dgName := range slice.SiteDeviceGroup {
// 		logger.ConfigLog.Debugf("dgName: %s", dgName)
// 		devGroupConfig := getDeviceGroupByName(dgName)
// 		if devGroupConfig == nil {
// 			logger.ConfigLog.Warnf("Device group not found: %s", dgName)
// 			continue
// 		}
// 		logger.AppLog.Debugf("slice=%s dg=%s: inputIMSIs=%d", slice.SliceName, dgName, len(devGroupConfig.Imsis))

// 		err = updateImsisConcurrently(devGroupConfig.Imsis, slice.SiteInfo.Plmn.Mcc, slice.SiteInfo.Plmn.Mnc, snssai,
// 			devGroupConfig.IpDomainExpanded.Dnn, devGroupConfig.IpDomainExpanded.UeDnnQos)

// 		if err != nil {
// 			logger.AppLog.Errorf("concurrent update failed for device group %s: %v", dgName, err)
// 			return http.StatusInternalServerError, err
// 		}

// 	}
// 	if err := cleanupDeviceGroups(slice, prevSlice); err != nil {
// 		return http.StatusInternalServerError, err
// 	}
// 	return http.StatusOK, nil
// }

// func updateImsisConcurrently(
// 	imsis []string,
// 	mcc string,
// 	mnc string,
// 	snssai *models.Snssai,
// 	dnn string,
// 	qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos,
// ) error {

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	sem := make(chan struct{}, factory.WebUIConfig.Configuration.Mongodb.ConcurrencyOps)
// 	errChan := make(chan error, 1)

// 	var wg sync.WaitGroup

// 	for _, imsi := range imsis {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		default:
// 		}

// 		wg.Add(1)
// 		sem <- struct{}{}
// 		logger.AppLog.Debugf("Starting update for IMSI %s", imsi)
// 		logger.AppLog.Debugf("len for pool operations is: %d", len(sem))

// 		go func(imsi string) {
// 			defer wg.Done()
// 			defer func() {
// 				<-sem
// 				logger.AppLog.Debugf("Finished update for IMSI %s", imsi)
// 				logger.AppLog.Debugf("len for pool operations is: %d", len(sem))
// 			}()

// 			// Si ya se canceló, no seguimos
// 			select {
// 			case <-ctx.Done():
// 				return
// 			default:
// 			}

// 			if err := updatePolicyAndProvisionedData(
// 				imsi,
// 				mcc,
// 				mnc,
// 				snssai,
// 				dnn,
// 				qos,
// 			); err != nil {

// 				logger.AppLog.Errorf("error %v", err)

// 				// Enviamos el error solo una vez
// 				select {
// 				case errChan <- err:
// 					cancel() // 🔥 cancela todas las demás gorutinas
// 				default:
// 				}
// 			}
// 		}(imsi)
// 	}

// 	wg.Wait()

// 	select {
// 	case err := <-errChan:
// 		return err
// 	default:
// 		return nil
// 	}
// }

func filterExistingIMSIsFromAuthDB(imsis []string) ([]string, error) {
	if len(imsis) == 0 {
		return nil, nil
	}
	logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: inputIMSIs=%d", len(imsis))
	if dbadapter.AuthDBClient == nil {
		// Keep behavior safe in tests/edge cases: assume all exist.
		logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: AuthDBClient is nil; returning input (safe default)")
		return slices.Clone(imsis), nil
	}

	ueIds := make([]string, 0, len(imsis))
	for _, imsi := range imsis {
		if strings.TrimSpace(imsi) == "" {
			continue
		}
		ueIds = append(ueIds, "imsi-"+imsi)
	}
	if len(ueIds) == 0 {
		return nil, nil
	}

	filter := bson.M{"ueId": bson.M{"$in": ueIds}}
	logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: querying authDB coll=%s with ueIds=%d", AuthSubsDataColl, len(ueIds))
	docs, err := dbadapter.AuthDBClient.RestfulAPIGetMany(AuthSubsDataColl, filter)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: authDB returned 0 docs")
		return nil, nil
	}
	logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: authDB returned docs=%d", len(docs))

	seen := make(map[string]struct{}, len(docs))
	for _, doc := range docs {
		ueId, _ := doc["ueId"].(string)
		imsi := strings.TrimPrefix(ueId, "imsi-")
		if imsi != "" {
			seen[imsi] = struct{}{}
		}
	}

	existing := make([]string, 0, len(seen))
	for _, imsi := range imsis {
		if _, ok := seen[imsi]; ok {
			existing = append(existing, imsi)
		}
	}
	logger.AppLog.Debugf("filterExistingIMSIsFromAuthDB: existingIMSIs=%d", len(existing))
	return existing, nil
}

const imsiBatchSize = 1000

func chunkStrings(in []string, size int) [][]string {
	if len(in) == 0 {
		return nil
	}
	if size <= 0 {
		return [][]string{in}
	}
	chunks := make([][]string, 0, (len(in)+size-1)/size)
	for start := 0; start < len(in); start += size {
		end := start + size
		if end > len(in) {
			end = len(in)
		}
		chunks = append(chunks, in[start:end])
	}
	return chunks
}

func cleanupDeviceGroups(slice, prevSlice configmodels.Slice) error {
	dgnames := getDeletedDeviceGroupsList(slice, prevSlice)
	for _, dgName := range dgnames {
		devGroupConfig := getDeviceGroupByName(dgName)
		if devGroupConfig == nil {
			logger.ConfigLog.Warnf("Device group not found during cleanup: %s", dgName)
			continue
		}
		// Compute with concurrency
		g, ctx := errgroup.WithContext(context.Background())
		g.SetLimit(factory.WebUIConfig.Configuration.Mongodb.ConcurrencyOps)
		for _, imsi := range devGroupConfig.Imsis {
			g.Go(func() error {
				// Verificar cancelación de contexto si hay error en otro lado
				if ctx.Err() != nil {
					return ctx.Err()
				}

				mcc := prevSlice.SiteInfo.Plmn.Mcc
				mnc := prevSlice.SiteInfo.Plmn.Mnc
				if err := removeSubscriberEntriesRelatedToDeviceGroups(mcc, mnc, imsi); err != nil {
					logger.ConfigLog.Errorf("Failed to remove subscriber for IMSI %s: %+v", imsi, err)
					return err
				}
				return nil
			})
		}
		// Esperar a que todos terminen
		if err := g.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func updatePolicyAndProvisionedData(imsi string, mcc string, mnc string, snssai *models.Snssai, dnn string, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) error {
	err := updateAmPolicyData(imsi)
	if err != nil {
		return fmt.Errorf("updateAmPolicyData failed: %w", err)
	}
	err = updateSmPolicyData(snssai, dnn, imsi)
	if err != nil {
		return fmt.Errorf("updateSmPolicyData failed: %w", err)
	}
	err = updateAmProvisionedData(snssai, qos, mcc, mnc, imsi)
	if err != nil {
		return fmt.Errorf("updateAmProvisionedData failed: %w", err)
	}
	err = updateSmProvisionedData(snssai, qos, mcc, mnc, dnn, imsi)
	if err != nil {
		return fmt.Errorf("updateSmProvisionedData failed: %w", err)
	}
	err = updateSmfSelectionProvisionedData(snssai, mcc, mnc, dnn, imsi)
	if err != nil {
		return fmt.Errorf("updateSmfSelectionProvisionedData failed: %w", err)
	}
	return nil
}

func updatePolicyAndProvisionedDataBatch(imsis []string, mcc string, mnc string, snssai *models.Snssai, dnn string, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) error {
	logger.AppLog.Debugf("updatePolicyAndProvisionedDataBatch: imsis=%d batchSize=%d mcc=%s mnc=%s dnn=%s", len(imsis), imsiBatchSize, mcc, mnc, dnn)
	return updatePoliciesAndProvisionedDatas(imsis, mcc, mnc, snssai, dnn, qos)
}

func updatePoliciesAndProvisionedDatas(imsis []string, mcc string, mnc string, snssai *models.Snssai, dnn string, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos) error {
	if len(imsis) == 0 {
		logger.AppLog.Debugf("updatePoliciesAndProvisionedDatas: no IMSIs; nothing to do")
		return nil
	}

	chunks := chunkStrings(imsis, imsiBatchSize)
	logger.AppLog.Debugf("updatePoliciesAndProvisionedDatas: totalIMSIs=%d chunks=%d batchSize=%d", len(imsis), len(chunks), imsiBatchSize)

	g := errgroup.Group{}
	g.SetLimit(factory.WebUIConfig.Configuration.Mongodb.ConcurrencyOps)

	for i, chunk := range chunks {
		g.Go(func() error {
			logger.AppLog.Debugf("updatePoliciesAndProvisionedDatas: processing chunk %d/%d (imsis=%d)", i+1, len(chunks), len(chunk))
			err := updateAmPolicyDatas(chunk)
			if err != nil {
				return fmt.Errorf("updateAmPolicyData failed (chunk %d/%d): %w", i+1, len(chunks), err)
			}
			err = updateSmPolicyDatas(snssai, dnn, chunk)
			if err != nil {
				return fmt.Errorf("updateSmPolicyData failed (chunk %d/%d): %w", i+1, len(chunks), err)
			}
			err = updateAmProvisionedDatas(snssai, qos, mcc, mnc, chunk)
			if err != nil {
				return fmt.Errorf("updateAmProvisionedData failed (chunk %d/%d): %w", i+1, len(chunks), err)
			}
			err = updateSmProvisionedDatas(snssai, qos, mcc, mnc, dnn, chunk)
			if err != nil {
				return fmt.Errorf("updateSmProvisionedData failed (chunk %d/%d): %w", i+1, len(chunks), err)
			}
			err = updateSmfSelectionProvisionedDatas(snssai, mcc, mnc, dnn, chunk)
			if err != nil {
				return fmt.Errorf("updateSmfSelectionProvisionedData failed (chunk %d/%d): %w", i+1, len(chunks), err)
			}
			logger.AppLog.Debugf("updatePoliciesAndProvisionedDatas: chunk %d/%d complete", i+1, len(chunks))

			logger.AppLog.Debugf("updatePoliciesAndProvisionedDatas: all chunks complete")
			return nil
		})
	}

	return g.Wait()
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src)+2)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func updateAmPolicyDatas(imsis []string) error {
	if len(imsis) == 0 {
		return nil
	}
	logger.AppLog.Debugf("updateAmPolicyDatas: coll=%s imsis=%d", AmPolicyDataColl, len(imsis))
	base := models.AmPolicyData{SubscCats: []string{"aether"}}
	baseDoc := configmodels.ToBsonM(base)

	filters := make([]primitive.M, 0, len(imsis))
	docs := make([]map[string]any, 0, len(imsis))
	for _, imsi := range imsis {
		ueId := "imsi-" + imsi
		doc := cloneMap(baseDoc)
		doc["ueId"] = ueId
		filters = append(filters, primitive.M{"ueId": ueId})
		docs = append(docs, doc)
	}

	logger.AppLog.Debugf("updateAmPolicyDatas: PutMany coll=%s docs=%d", AmPolicyDataColl, len(docs))
	if err := dbadapter.CommonDBClient.RestfulAPIPutMany(AmPolicyDataColl, filters, docs); err != nil {
		logger.AppLog.Errorf("failed to batch update AM Policy Data for %d IMSIs: %+v", len(imsis), err)
		return err
	}
	return nil
}

func updateSmPolicyDatas(snssai *models.Snssai, dnn string, imsis []string) error {
	if len(imsis) == 0 {
		return nil
	}
	logger.AppLog.Debugf("updateSmPolicyDatas: coll=%s imsis=%d dnn=%s", SmPolicyDataColl, len(imsis), dnn)
	var smPolicyData models.SmPolicyData
	var smPolicySnssaiData models.SmPolicySnssaiData
	dnnData := map[string]models.SmPolicyDnnData{dnn: {Dnn: dnn}}
	smPolicySnssaiData.Snssai = snssai
	smPolicySnssaiData.SmPolicyDnnData = dnnData
	smPolicyData.SmPolicySnssaiData = make(map[string]models.SmPolicySnssaiData)
	smPolicyData.SmPolicySnssaiData[SnssaiModelsToHex(*snssai)] = smPolicySnssaiData
	baseDoc := configmodels.ToBsonM(smPolicyData)

	filters := make([]primitive.M, 0, len(imsis))
	docs := make([]map[string]any, 0, len(imsis))
	for _, imsi := range imsis {
		ueId := "imsi-" + imsi
		doc := cloneMap(baseDoc)
		doc["ueId"] = ueId
		filters = append(filters, primitive.M{"ueId": ueId})
		docs = append(docs, doc)
	}

	logger.AppLog.Debugf("updateSmPolicyDatas: PutMany coll=%s docs=%d", SmPolicyDataColl, len(docs))
	if err := dbadapter.CommonDBClient.RestfulAPIPutMany(SmPolicyDataColl, filters, docs); err != nil {
		logger.AppLog.Errorf("failed to batch update SM Policy Data for %d IMSIs: %+v", len(imsis), err)
		return err
	}
	return nil
}

func updateAmProvisionedDatas(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc string, mnc string, imsis []string) error {
	if len(imsis) == 0 {
		return nil
	}
	logger.AppLog.Debugf("updateAmProvisionedDatas: coll=%s imsis=%d mcc=%s mnc=%s", AmDataColl, len(imsis), mcc, mnc)
	plmn := mcc + mnc
	amData := models.AccessAndMobilitySubscriptionData{
		Gpsis:            []string{"msisdn-0900000000"},
		Nssai:            &models.Nssai{DefaultSingleNssais: []models.Snssai{*snssai}, SingleNssais: []models.Snssai{*snssai}},
		SubscribedUeAmbr: &models.AmbrRm{Downlink: ConvertToString(uint64(qos.DnnMbrDownlink)), Uplink: ConvertToString(uint64(qos.DnnMbrUplink))},
	}
	baseDoc := configmodels.ToBsonM(amData)

	filters := make([]primitive.M, 0, len(imsis))
	docs := make([]map[string]any, 0, len(imsis))
	for _, imsi := range imsis {
		ueId := "imsi-" + imsi
		doc := cloneMap(baseDoc)
		doc["ueId"] = ueId
		doc["servingPlmnId"] = plmn
		filters = append(filters, primitive.M{
			"ueId": ueId,
			"$or":  []bson.M{{"servingPlmnId": plmn}, {"servingPlmnId": bson.M{"$exists": false}}},
		})
		docs = append(docs, doc)
	}

	logger.AppLog.Debugf("updateAmProvisionedDatas: PutMany coll=%s docs=%d", AmDataColl, len(docs))
	if err := dbadapter.CommonDBClient.RestfulAPIPutMany(AmDataColl, filters, docs); err != nil {
		logger.AppLog.Errorf("failed to batch update AM provisioned Data for %d IMSIs: %+v", len(imsis), err)
		return err
	}
	return nil
}

func updateSmProvisionedDatas(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc string, mnc string, dnn string, imsis []string) error {
	if len(imsis) == 0 {
		return nil
	}
	logger.AppLog.Debugf("updateSmProvisionedDatas: coll=%s imsis=%d mcc=%s mnc=%s dnn=%s", SmDataColl, len(imsis), mcc, mnc, dnn)
	plmn := mcc + mnc
	smData := models.SessionManagementSubscriptionData{
		SingleNssai: snssai,
		DnnConfigurations: map[string]models.DnnConfiguration{
			dnn: {
				PduSessionTypes: &models.PduSessionTypes{DefaultSessionType: models.PduSessionType_IPV4, AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4}},
				SscModes:        &models.SscModes{DefaultSscMode: models.SscMode__1, AllowedSscModes: []models.SscMode{"SSC_MODE_2", "SSC_MODE_3"}},
				SessionAmbr:     &models.Ambr{Downlink: ConvertToString(uint64(qos.DnnMbrDownlink)), Uplink: ConvertToString(uint64(qos.DnnMbrUplink))},
				Var5gQosProfile: &models.SubscribedDefaultQos{Var5qi: 9, Arp: &models.Arp{PriorityLevel: 8}, PriorityLevel: 8},
			},
		},
	}
	baseDoc := configmodels.ToBsonM(smData)

	filters := make([]primitive.M, 0, len(imsis))
	docs := make([]map[string]any, 0, len(imsis))
	for _, imsi := range imsis {
		ueId := "imsi-" + imsi
		doc := cloneMap(baseDoc)
		doc["ueId"] = ueId
		doc["servingPlmnId"] = plmn
		filters = append(filters, primitive.M{"ueId": ueId, "servingPlmnId": plmn})
		docs = append(docs, doc)
	}

	logger.AppLog.Debugf("updateSmProvisionedDatas: PutMany coll=%s docs=%d", SmDataColl, len(docs))
	if err := dbadapter.CommonDBClient.RestfulAPIPutMany(SmDataColl, filters, docs); err != nil {
		logger.AppLog.Errorf("failed to batch update SM provisioned Data for %d IMSIs: %+v", len(imsis), err)
		return err
	}
	return nil
}

func updateSmfSelectionProvisionedDatas(snssai *models.Snssai, mcc string, mnc string, dnn string, imsis []string) error {
	if len(imsis) == 0 {
		return nil
	}
	logger.AppLog.Debugf("updateSmfSelectionProvisionedDatas: coll=%s imsis=%d mcc=%s mnc=%s dnn=%s", SmfSelDataColl, len(imsis), mcc, mnc, dnn)
	plmn := mcc + mnc
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			SnssaiModelsToHex(*snssai): {DnnInfos: []models.DnnInfo{{Dnn: dnn}}},
		},
	}
	baseDoc := configmodels.ToBsonM(smfSelData)

	filters := make([]primitive.M, 0, len(imsis))
	docs := make([]map[string]any, 0, len(imsis))
	for _, imsi := range imsis {
		ueId := "imsi-" + imsi
		doc := cloneMap(baseDoc)
		doc["ueId"] = ueId
		doc["servingPlmnId"] = plmn
		filters = append(filters, primitive.M{"ueId": ueId, "servingPlmnId": plmn})
		docs = append(docs, doc)
	}

	logger.AppLog.Debugf("updateSmfSelectionProvisionedDatas: PutMany coll=%s docs=%d", SmfSelDataColl, len(docs))
	if err := dbadapter.CommonDBClient.RestfulAPIPutMany(SmfSelDataColl, filters, docs); err != nil {
		logger.AppLog.Errorf("failed to batch update SMF selection provisioned data for %d IMSIs: %+v", len(imsis), err)
		return err
	}
	return nil
}

func updateAmPolicyData(imsi string) error {
	var amPolicy models.AmPolicyData
	amPolicy.SubscCats = append(amPolicy.SubscCats, "aether")
	amPolicyDatBsonA := configmodels.ToBsonM(amPolicy)
	amPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(AmPolicyDataColl, filter, amPolicyDatBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to update AM Policy Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to update AM Policy Data for IMSI %s", imsi)
	return nil
}

func updateSmPolicyData(snssai *models.Snssai, dnn string, imsi string) error {
	var smPolicyData models.SmPolicyData
	var smPolicySnssaiData models.SmPolicySnssaiData
	dnnData := map[string]models.SmPolicyDnnData{
		dnn: {
			Dnn: dnn,
		},
	}
	// smpolicydata
	smPolicySnssaiData.Snssai = snssai
	smPolicySnssaiData.SmPolicyDnnData = dnnData
	smPolicyData.SmPolicySnssaiData = make(map[string]models.SmPolicySnssaiData)
	smPolicyData.SmPolicySnssaiData[SnssaiModelsToHex(*snssai)] = smPolicySnssaiData
	smPolicyDatBsonA := configmodels.ToBsonM(smPolicyData)
	smPolicyDatBsonA["ueId"] = "imsi-" + imsi
	filter := bson.M{"ueId": "imsi-" + imsi}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(SmPolicyDataColl, filter, smPolicyDatBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to update SM Policy Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to update SM Policy Data for IMSI %s", imsi)
	return nil
}

func updateAmProvisionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, imsi string) error {
	amData := models.AccessAndMobilitySubscriptionData{
		Gpsis: []string{
			"msisdn-0900000000",
		},
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{*snssai},
			SingleNssais:        []models.Snssai{*snssai},
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: ConvertToString(uint64(qos.DnnMbrDownlink)),
			Uplink:   ConvertToString(uint64(qos.DnnMbrUplink)),
		},
	}
	amDataBsonA := configmodels.ToBsonM(amData)
	amDataBsonA["ueId"] = "imsi-" + imsi
	amDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{
		"ueId": "imsi-" + imsi,
		"$or": []bson.M{
			{"servingPlmnId": mcc + mnc},
			{"servingPlmnId": bson.M{"$exists": false}},
		},
	}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(AmDataColl, filter, amDataBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to update AM provisioned Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to update AM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmProvisionedData(snssai *models.Snssai, qos *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos, mcc, mnc, dnn, imsi string) error {
	smData := models.SessionManagementSubscriptionData{
		SingleNssai: snssai,
		DnnConfigurations: map[string]models.DnnConfiguration{
			dnn: {
				PduSessionTypes: &models.PduSessionTypes{
					DefaultSessionType:  models.PduSessionType_IPV4,
					AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
				},
				SscModes: &models.SscModes{
					DefaultSscMode: models.SscMode__1,
					AllowedSscModes: []models.SscMode{
						"SSC_MODE_2",
						"SSC_MODE_3",
					},
				},
				SessionAmbr: &models.Ambr{
					Downlink: ConvertToString(uint64(qos.DnnMbrDownlink)),
					Uplink:   ConvertToString(uint64(qos.DnnMbrUplink)),
				},
				Var5gQosProfile: &models.SubscribedDefaultQos{
					Var5qi: 9,
					Arp: &models.Arp{
						PriorityLevel: 8,
					},
					PriorityLevel: 8,
				},
			},
		},
	}
	smDataBsonA := configmodels.ToBsonM(smData)
	smDataBsonA["ueId"] = "imsi-" + imsi
	smDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(SmDataColl, filter, smDataBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to update SM provisioned Data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("updated SM provisioned Data for IMSI %s", imsi)
	return nil
}

func updateSmfSelectionProvisionedData(snssai *models.Snssai, mcc, mnc, dnn, imsi string) error {
	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			SnssaiModelsToHex(*snssai): {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: dnn,
					},
				},
			},
		},
	}
	smfSelecDataBsonA := configmodels.ToBsonM(smfSelData)
	smfSelecDataBsonA["ueId"] = "imsi-" + imsi
	smfSelecDataBsonA["servingPlmnId"] = mcc + mnc
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}
	_, err := dbadapter.CommonDBClient.RestfulAPIPost(SmfSelDataColl, filter, smfSelecDataBsonA)
	if err != nil {
		logger.AppLog.Errorf("failed to update SMF selection provisioned data for IMSI %s: %+v", imsi, err)
		return err
	}
	logger.AppLog.Debugf("updated SMF selection provisioned data for IMSI %s", imsi)
	return nil
}

func SnssaiModelsToHex(snssai models.Snssai) string {
	sst := fmt.Sprintf("%02x", snssai.Sst)
	return sst + snssai.Sd
}

func ConvertToString(val uint64) string {
	var mbVal, gbVal, kbVal uint64
	kbVal = val / 1000
	mbVal = val / 1000000
	gbVal = val / 1000000000
	var retStr string
	if gbVal != 0 {
		retStr = strconv.FormatUint(gbVal, 10) + " Gbps"
	} else if mbVal != 0 {
		retStr = strconv.FormatUint(mbVal, 10) + " Mbps"
	} else if kbVal != 0 {
		retStr = strconv.FormatUint(kbVal, 10) + " Kbps"
	} else {
		retStr = strconv.FormatUint(val, 10) + " bps"
	}

	return retStr
}

func getSlices() []*configmodels.Slice {
	rawSlices, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, nil)
	if errGetMany != nil {
		logger.AppLog.Warnln(errGetMany)
	}
	var slices []*configmodels.Slice
	for _, rawSlice := range rawSlices {
		var sliceData configmodels.Slice
		err := json.Unmarshal(configmodels.MapToByte(rawSlice), &sliceData)
		if err != nil {
			logger.AppLog.Errorf("could not unmarshall slice %+v", rawSlice)
		}
		slices = append(slices, &sliceData)
	}
	return slices
}

func getSliceByName(name string) *configmodels.Slice {
	filter := bson.M{"slice-name": name}
	sliceDataInterface, errGetOne := dbadapter.CommonDBClient.RestfulAPIGetOne(sliceDataColl, filter)
	if errGetOne != nil {
		logger.AppLog.Warnln(errGetOne)
		return nil
	}
	var sliceData configmodels.Slice
	err := json.Unmarshal(configmodels.MapToByte(sliceDataInterface), &sliceData)
	if err != nil {
		logger.AppLog.Errorf("could not unmarshall slice %+v", sliceDataInterface)
		return nil
	}
	return &sliceData
}

func handleNetworkSliceDelete(sliceName string) error {
	prevSlice := getSliceByName(sliceName)
	filter := bson.M{"slice-name": sliceName}
	err := dbadapter.CommonDBClient.RestfulAPIDeleteOne(sliceDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed to delete slice data for %+v: %+v", sliceName, err)
		return err
	}
	// slice is nil as it is deleted
	if err = syncSubscribersOnSliceDelete(nil, prevSlice); err != nil {
		logger.WebUILog.Errorf("failed to cleanup subscriber entries related to device groups %+v: %+v", sliceName, err)
		return err
	}
	logger.AppLog.Debugf("succeeded to delete slice data for %s", sliceName)
	if factory.WebUIConfig.Configuration.SendPebbleNotifications {
		err = sendPebbleNotification("aetherproject.org/webconsole/networkslice/delete")
		if err != nil {
			logger.ConfigLog.Warnf("sending Pebble notification failed: %s. continuing silently", err.Error())
		}
	}
	return nil
}

func getDeletedDeviceGroupsList(slice, prevSlice configmodels.Slice) []string {
	if len(prevSlice.SiteDeviceGroup) == 0 {
		return nil
	}
	if len(slice.SiteDeviceGroup) == 0 {
		return slices.Clone(prevSlice.SiteDeviceGroup)
	}

	var deleted []string
	for _, pdgName := range prevSlice.SiteDeviceGroup {
		if !slices.Contains(slice.SiteDeviceGroup, pdgName) {
			deleted = append(deleted, pdgName)
		}
	}
	return deleted
}
