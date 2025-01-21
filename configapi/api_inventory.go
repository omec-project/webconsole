// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	logger.WebUILog.Infoln("get all gNBs")

	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
	rawGnbs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(configmodels.GnbDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorln(errGetMany)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve gNBs"})
		return
	}

	for _, rawGnb := range rawGnbs {
		var gnbData configmodels.Gnb
		err := json.Unmarshal(configmodels.MapToByte(rawGnb), &gnbData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshal gNB %v", rawGnb)
		}
		gnbs = append(gnbs, &gnbData)
	}
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
// @Failure      400  {object}  nil  "Failed to delete the gNB"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Router       /config/v1/inventory/gnb/{gnb-name}  [delete]
func DeleteGnb(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := handleDeleteGnb(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func handlePostGnb(c *gin.Context) error {
	gnbName, exists := c.Params.Get("gnb-name")
	if !exists {
		errorMessage := "post gNB request is missing gnb-name"
		logger.ConfigLog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.ConfigLog.Infof("received gNB %v", gnbName)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return fmt.Errorf("invalid header")
	}
	var postGnbRequest configmodels.PostGnbRequest
	err := c.ShouldBindJSON(&postGnbRequest)
	if err != nil {
		logger.ConfigLog.Errorf("err %v", err)
		return fmt.Errorf("invalid JSON format")
	}
	if postGnbRequest.Tac == "" {
		errorMessage := "post gNB request body is missing tac"
		logger.ConfigLog.Errorln(errorMessage)
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
	logger.ConfigLog.Infof("successfully added gNB [%v] to config channel", gnbName)
	return nil
}

func handleDeleteGnb(c *gin.Context) error {
	gnbName, exists := c.Params.Get("gnb-name")
	if !exists {
		errorMessage := "delete gNB request is missing gnb-name"
		logger.ConfigLog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.ConfigLog.Infof("received delete gNB %v request", gnbName)
	msg := configmodels.ConfigMessage{
		MsgType:   configmodels.Inventory,
		MsgMethod: configmodels.Delete_op,
		GnbName:   gnbName,
	}
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully added gNB [%v] with delete_op to config channel", gnbName)
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
	logger.WebUILog.Infoln("get all UPFs")

	var upfs []*configmodels.Upf
	upfs = make([]*configmodels.Upf, 0)
	rawUpfs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(configmodels.UpfDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorln(errGetMany)
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
	c.JSON(http.StatusOK, upfs)
}

// PostUpf godoc
//
// @Description  Create a new UPF
// @Tags         UPFs
// @Produce      json
// @Param        upf  body  configmodels.PostUpfRequest  true  "Hostname and port of the UPF to create"
// @Security     BearerAuth
// @Success      201  {object}  nil  "UPF successfully created"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Error creating UPF"
// @Router       /config/v1/inventory/upf/  [post]
func PostUpf(c *gin.Context) {
	logger.WebUILog.Infoln("received a POST UPF request")
	var postUpfParams configmodels.PostUpfRequest
	err := c.ShouldBindJSON(&postUpfParams)
	if err != nil {
		logger.WebUILog.Errorw("invalid UPF POST input parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if postUpfParams.Hostname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UPF hostname must be provided"})
		return
	}
	if _, err := strconv.Atoi(postUpfParams.Port); err != nil {
		errorMessage := "UPF port cannot be converted to integer or it was not provided"
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	upf := configmodels.Upf(postUpfParams)
	if err = handleUpfTransaction(c.Request.Context(), upf, postUpfOperation); err != nil {
		if strings.Contains(err.Error(), "E11000") {
			logger.WebUILog.Errorw("duplicate hostname found:", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "UPF already exists"})
			return
		}
		logger.WebUILog.Errorw("failed to create UPF", "hostname", postUpfParams.Hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed POST UPF %v request", postUpfParams.Hostname)
	c.JSON(http.StatusCreated, gin.H{})
}

// PutUpf godoc
//
// @Description  Create or update a UPF
// @Tags         UPFs
// @Produce      json
// @Param        upf-hostname   path    string                       true    "Name of the UPF to update"
// @Param        port           body    configmodels.PutUpfRequest   true    "Port of the UPF to update"
// @Security     BearerAuth
// @Success      200  {object}  nil  "UPF successfully updated"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Error updating UPF"
// @Router       /config/v1/inventory/upf/{upf-hostname}  [put]
func PutUpf(c *gin.Context) {
	logger.WebUILog.Infoln("received a PUT UPF request")
	hostname, exists := c.Params.Get("upf-hostname")
	if !exists {
		errorMessage := "put UPF request is missing path param `upf-hostname`"
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	var putUpfParams configmodels.PutUpfRequest
	err := c.ShouldBindJSON(&putUpfParams)
	if err != nil {
		logger.WebUILog.Errorw("invalid UPF PUT input parameters", "hostname", hostname, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if _, err := strconv.Atoi(putUpfParams.Port); err != nil {
		errorMessage := "UPF port cannot be converted to integer or it was not provided"
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	putUpf := configmodels.Upf{
		Hostname: hostname,
		Port:     putUpfParams.Port,
	}
	if err := handleUpfTransaction(c.Request.Context(), putUpf, putUpfOperation); err != nil {
		logger.WebUILog.Errorw("failed to PUT UPF", "hostname", hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT UPF"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func postUpfOperation(upf configmodels.Upf, sc mongo.SessionContext) error {
	filter := bson.M{"hostname": upf.Hostname}
	upfDataBson := configmodels.ToBsonM(upf)
	return dbadapter.CommonDBClient.RestfulAPIPostMany(configmodels.UpfDataColl, filter, []interface{}{upfDataBson})
}

func putUpfOperation(upf configmodels.Upf, sc mongo.SessionContext) error {
	filter := bson.M{"hostname": upf.Hostname}
	upfDataBson := configmodels.ToBsonM(upf)
	_, err := dbadapter.CommonDBClient.RestfulAPIPutOneWithContext(configmodels.UpfDataColl, filter, upfDataBson, sc)
	return err
}

func handleUpfTransaction(ctx context.Context, upf configmodels.Upf, operation func(configmodels.Upf, mongo.SessionContext) error) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err := operation(upf, sc); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return err
		}
		patchJSON := []byte(fmt.Sprintf(`[
			{
				"op": "replace",
				"path": "/site-info/upf",
				"value": {
					"upf-name": "%s",
					"upf-port": "%s"
				}
			}
		]`, upf.Hostname, upf.Port))
		err = updateUpfInNetworkSlices(upf.Hostname, patchJSON, sc)
		if err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

// DeleteUpf godoc
//
// @Description  Delete an existing UPF
// @Tags         UPFs
// @Produce      json
// @Param        upf-hostname    path    string    true    "Name of the UPF"
// @Security     BearerAuth
// @Success      200  {object}  nil  "UPF deleted"
// @Failure      400  {object}  nil  "Failed to delete the UPF"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Router       /config/v1/inventory/upf/{upf-hostname}  [delete]
func DeleteUpf(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := handleDeleteUpf(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func handleDeleteUpf(c *gin.Context) error {
	upfHostname, exists := c.Params.Get("upf-hostname")
	if !exists {
		errorMessage := "delete UPF request is missing upf-hostname"
		logger.ConfigLog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.ConfigLog.Infof("received delete UPF %v", upfHostname)
	msg := configmodels.ConfigMessage{
		MsgType:     configmodels.Inventory,
		MsgMethod:   configmodels.Delete_op,
		UpfHostname: upfHostname,
	}
	configChannel <- &msg
	logger.ConfigLog.Infof("successfully added UPF [%v] with delete_op to config channel", upfHostname)
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
		err = dbadapter.CommonDBClient.RestfulAPIJSONPatchWithContext(sliceDataColl, filterBySliceName, patchJSON, context)
		if err != nil {
			return err
		}
	}
	return nil
}
