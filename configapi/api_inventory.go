// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	gnbDataColl = "webconsoleData.snapshots.gnbData"
	upfDataColl = "webconsoleData.snapshots.upfData"
)

func setInventoryCorsHeader(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE")
}

// GetGnbs godoc
//
// @Description Return the list of gNBs
// @Tags        gNBs
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   configmodels.Gnb  "List of gNBs"
// @Failure     401  {object}  nil               "Authorization failed"
// @Failure     403  {object}  nil               "Forbidden"
// @Failure     500  {object}  nil               "Error retrieving gNBs"
// @Router      /config/v1/inventory/gnb  [get]
func GetGnbs(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a GET gNBs request")
	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
	rawGnbs, err := dbadapter.CommonDBClient.RestfulAPIGetMany(gnbDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorw("failed to retrieve gNBs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve gNBs"})
		return
	}

	for _, rawGnb := range rawGnbs {
		var gnbData configmodels.Gnb
		err = json.Unmarshal(configmodels.MapToByte(rawGnb), &gnbData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshal gNB %v", rawGnb)
		}
		gnbs = append(gnbs, &gnbData)
	}
	logger.WebUILog.Infoln("successfully executed GET gNBs request")
	c.JSON(http.StatusOK, gnbs)
}

// PostGnb godoc
//
// @Description Create a new gNB
// @Tags        gNBs
// @Produce     json
// @Param       gnb-name    path    string                         true    "Name of the gNB"
// @Param       tac         body    configmodels.PostGnbRequest    true    "TAC of the gNB"
// @Security    BearerAuth
// @Success     200  {object}  nil  "gNB created"
// @Failure     400  {object}  nil  "Failed to create the gNB"
// @Failure     401  {object}  nil  "Authorization failed"
// @Failure     403  {object}  nil  "Forbidden"
// @Router      /config/v1/inventory/gnb/{gnb-name}  [post]
func PostGnb(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := handlePostGnb(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// DeleteGnb godoc
//
// @Description  Delete an existing gNB
// @Tags         gNBs
// @Produce      json
// @Param        gnb-name    path    string    true    "Name of the gNB"
// @Security     BearerAuth
// @Success      200  {object}  nil  "gNB deleted"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Failed to delete gNB"
// @Router       /config/v1/inventory/gnb/{gnb-name}  [delete]
func DeleteGnb(c *gin.Context) {
	logger.WebUILog.Infoln("received a DELETE gNB request")
	setInventoryCorsHeader(c)
	gnbName, exists := c.Params.Get("gnb-name")
	if !exists {
		errorMessage := "delete gNB request is missing path param `gnb-name`"
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	filter := bson.M{"name": gnbName}
	err := handleDeleteGnbTransaction(c.Request.Context(), filter, gnbName)
	if err != nil {
		logger.WebUILog.Errorw("failed to delete gNB", "gnbName", gnbName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete gNB"})
		return
	}
	logger.WebUILog.Infof("successfully executed DELETE gNB %v request", gnbName)
	c.JSON(http.StatusOK, gin.H{})
}

func handleDeleteGnbTransaction(ctx context.Context, filter bson.M, gnbName string) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, gnbDataColl, filter); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to delete gNB from collection: %w", err)
		}
		if err = updateGnbInNetworkSlices(gnbName, sc); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

func handlePostGnb(c *gin.Context) error {
	gnbName, exists := c.Params.Get("gnb-name")
	if !exists {
		errorMessage := "post gNB request is missing gnb-name"
		logger.WebUILog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.WebUILog.Infof("received a POST gNB %v request", gnbName)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return fmt.Errorf("invalid header")
	}
	var postGnbRequest configmodels.PostGnbRequest
	err := c.ShouldBindJSON(&postGnbRequest)
	if err != nil {
		logger.WebUILog.Errorf("err %v", err)
		return fmt.Errorf("invalid JSON format")
	}
	if postGnbRequest.Tac == "" {
		errorMessage := "post gNB request body is missing tac"
		logger.WebUILog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	postGnb := configmodels.Gnb{
		Name: gnbName,
		Tac:  postGnbRequest.Tac,
	}
	msg := configmodels.ConfigMessage{
		MsgType:   configmodels.Inventory,
		MsgMethod: configmodels.Post_op,
		Gnb:       &postGnb,
	}
	configChannel <- &msg
	logger.WebUILog.Infof("successfully added gNB [%v] to config channel", gnbName)
	return nil
}

// GetUpfs godoc
//
// @Description  Return the list of UPFs
// @Tags         UPFs
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   configmodels.Upf  "List of UPFs"
// @Failure      401  {object}  nil               "Authorization failed"
// @Failure      403  {object}  nil               "Forbidden"
// @Failure      500  {object}  nil               "Error retrieving UPFs"
// @Router       /config/v1/inventory/upf  [get]
func GetUpfs(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a GET UPFs request")
	var upfs []*configmodels.Upf
	upfs = make([]*configmodels.Upf, 0)
	rawUpfs, err := dbadapter.CommonDBClient.RestfulAPIGetMany(upfDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorw("failed to retrieve UPFs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve UPFs"})
		return
	}

	for _, rawUpf := range rawUpfs {
		var upfData configmodels.Upf
		err := json.Unmarshal(configmodels.MapToByte(rawUpf), &upfData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshal UPF %v", rawUpf)
		}
		upfs = append(upfs, &upfData)
	}
	logger.WebUILog.Infoln("successfully executed GET UPFs request")
	c.JSON(http.StatusOK, upfs)
}

// PostUpf godoc
//
// @Description  Create a new UPF
// @Tags         UPFs
// @Produce      json
// @Param        upf-hostname   path    string                         true    "Name of the UPF"
// @Param        port           body    configmodels.PostUpfRequest    true    "Port of the UPF"
// @Security     BearerAuth
// @Success      200  {object}  nil  "UPF created"
// @Failure      400  {object}  nil  "Failed to create the UPF"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Router       /config/v1/inventory/upf/{upf-hostname}  [post]
func PostUpf(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := handlePostUpf(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// DeleteUpf godoc
//
// @Description  Delete an existing UPF
// @Tags         UPFs
// @Produce      json
// @Param        upf-hostname    path    string    true    "Name of the UPF"
// @Security     BearerAuth
// @Success      200  {object}  nil  "UPF deleted"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Failed to delete UPF"
// @Router       /config/v1/inventory/upf/{upf-hostname}  [delete]
func DeleteUpf(c *gin.Context) {
	logger.WebUILog.Infoln("received a DELETE UPF request")
	setInventoryCorsHeader(c)
	hostname, exists := c.Params.Get("upf-hostname")
	if !exists {
		errorMessage := "delete gNB request is missing path param `upf-hostname`"
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	filter := bson.M{"hostname": hostname}
	err := handleDeleteUpfTransaction(c.Request.Context(), filter, hostname)
	if err != nil {
		logger.WebUILog.Errorw("failed to delete UPF", "hostname", hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed DELETE UPF request for hostname: %v", hostname)
	c.JSON(http.StatusOK, gin.H{})
}

func handleDeleteUpfTransaction(ctx context.Context, filter bson.M, hostname string) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, upfDataColl, filter); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return err
		}
		patchJSON := []byte(`[{"op": "remove", "path": "/site-info/upf"}]`)
		if err = updateUpfInNetworkSlices(hostname, patchJSON, sc); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

func handlePostUpf(c *gin.Context) error {
	upfHostname, exists := c.Params.Get("upf-hostname")
	if !exists {
		errorMessage := "post UPF request is missing upf-hostname"
		logger.WebUILog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.WebUILog.Infof("received a POST UPF %v request", upfHostname)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return fmt.Errorf("invalid header")
	}
	var postUpfRequest configmodels.PostUpfRequest
	err := c.ShouldBindJSON(&postUpfRequest)
	if err != nil {
		logger.WebUILog.Errorf("err %v", err)
		return fmt.Errorf("invalid JSON format")
	}
	if postUpfRequest.Port == "" {
		errorMessage := "post UPF request body is missing port"
		logger.WebUILog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	postUpf := configmodels.Upf{
		Hostname: upfHostname,
		Port:     postUpfRequest.Port,
	}
	msg := configmodels.ConfigMessage{
		MsgType:   configmodels.Inventory,
		MsgMethod: configmodels.Post_op,
		Upf:       &postUpf,
	}
	configChannel <- &msg
	logger.WebUILog.Infof("successfully added UPF [%v] to config channel", upfHostname)
	return nil
}

func updateGnbInNetworkSlices(gnbName string, context context.Context) error {
	filterByGnb := bson.M{
		"site-info.gNodeBs": bson.M{
			"$elemMatch": bson.M{"name": gnbName},
		},
	}
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filterByGnb)
	if err != nil {
		return fmt.Errorf("failed to fetch network slices: %w", err)
	}
	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			return fmt.Errorf("error unmarshaling network slice: %v", err)
		}
		filteredGNodeBs := []configmodels.SliceSiteInfoGNodeBs{}
		for _, gnb := range networkSlice.SiteInfo.GNodeBs {
			if gnb.Name != gnbName {
				filteredGNodeBs = append(filteredGNodeBs, gnb)
			}
		}
		filteredGNodeBsJSON, err := json.Marshal(filteredGNodeBs)
		if err != nil {
			return fmt.Errorf("error marshaling GNodeBs: %v", err)
		}
		patchJSON := []byte(
			fmt.Sprintf(`[{"op": "replace", "path": "/site-info/gNodeBs", "value": %s}]`,
				string(filteredGNodeBsJSON)),
		)
		filterBySliceName := bson.M{"slice-name": networkSlice.SliceName}
		err = dbadapter.CommonDBClient.RestfulAPIJSONPatchWithContext(context, sliceDataColl, filterBySliceName, patchJSON)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateUpfInNetworkSlices(hostname string, patchJSON []byte, context context.Context) error {
	filterByUpf := bson.M{"site-info.upf.upf-name": hostname}
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filterByUpf)
	if err != nil {
		return fmt.Errorf("failed to fetch network slices: %w", err)
	}
	for _, rawNetworkSlice := range rawNetworkSlices {
		sliceName, ok := rawNetworkSlice["slice-name"].(string)
		if !ok {
			return fmt.Errorf("invalid slice-name in network slice: %v", rawNetworkSlice)
		}
		filterBySliceName := bson.M{"slice-name": sliceName}
		err = dbadapter.CommonDBClient.RestfulAPIJSONPatchWithContext(context, sliceDataColl, filterBySliceName, patchJSON)
		if err != nil {
			return err
		}
	}
	return nil
}
