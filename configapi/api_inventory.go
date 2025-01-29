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
	logger.WebUILog.Infoln("received a GET gNBs request")
	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
	rawGnbs, err := dbadapter.CommonDBClient.RestfulAPIGetMany(configmodels.GnbDataColl, bson.M{})
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
// @Param       gnb    body    configmodels.PostGnbRequest    true    "Name and TAC of the gNB"
// @Security    BearerAuth
// @Success     201  {object}  nil  "gNB sucessfully created"
// @Failure     400  {object}  nil  "Bad request"
// @Failure     401  {object}  nil  "Authorization failed"
// @Failure     403  {object}  nil  "Forbidden"
// @Failure     500  {object}  nil  "Error creating gNB"
// @Router      /config/v1/inventory/gnb  [post]
func PostGnb(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a POST gNB request")
	var postGnbParams configmodels.PostGnbRequest
	if err := c.ShouldBindJSON(&postGnbParams); err != nil {
		logger.WebUILog.Errorw("invalid UPF gNB input parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if !isValidName(postGnbParams.Name) {
		errorMessage := fmt.Sprintf("invalid gNB name '%s'. Name needs to match the following regular expression: %s", postGnbParams.Name, NAME_PATTERN)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	if !isValidGnbTac(postGnbParams.Tac) {
		errorMessage := fmt.Sprintf("invalid gNB TAC '%v'. TAC must be a numeric string within the range [1, 16777215]", postGnbParams.Tac)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	gnb := configmodels.Gnb(postGnbParams)
	patchJSON := []byte{}
	if err := executeGnbTransaction(c.Request.Context(), gnb, patchJSON, postGnbOperation); err != nil {
		if strings.Contains(err.Error(), "E11000") {
			logger.WebUILog.Errorw("duplicate gNB name found:", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "gNB already exists"})
			return
		}
		logger.WebUILog.Errorw("failed to create gNB", "name", postGnbParams.Name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create gNB"})
		return
	}
	logger.WebUILog.Infof("successfully executed POST gNB %v request", postGnbParams.Name)
	c.JSON(http.StatusCreated, gin.H{})
}

// PutGnb godoc
//
// @Description Create or update a gNB
// @Tags        gNBs
// @Produce     json
// @Param       gnb-name    path    string                        true    "Name of the gNB"
// @Param       tac         body    configmodels.PutGnbRequest    true    "TAC of the gNB"
// @Security    BearerAuth
// @Success     201  {object}  nil  "gNB sucessfully created"
// @Failure     400  {object}  nil  "Bad request"
// @Failure     401  {object}  nil  "Authorization failed"
// @Failure     403  {object}  nil  "Forbidden"
// @Failure     500  {object}  nil  "Error updating gNB"
// @Router      /config/v1/inventory/gnb/{gnb-name}  [put]
func PutGnb(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a PUT gNB request")
	gnbName, _ := c.Params.Get("gnb-name")
	if !isValidName(gnbName) {
		errorMessage := fmt.Sprintf("invalid gNB name '%s'. Name needs to match the following regular expression: %s", gnbName, NAME_PATTERN)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	var putGnbParams configmodels.PutGnbRequest
	if err := c.ShouldBindJSON(&putGnbParams); err != nil {
		logger.WebUILog.Errorw("invalid gNB PUT input parameters", "name", gnbName, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if !isValidGnbTac(putGnbParams.Tac) {
		errorMessage := fmt.Sprintf("invalid gnb TAC '%v'. TAC must be a numeric string within the range [1, 16777215]", putGnbParams.Tac)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	putGnb := configmodels.Gnb{
		Name: gnbName,
		Tac:  putGnbParams.Tac,
	}
	patchJSON := []byte{}
	if err := executeGnbTransaction(c.Request.Context(), putGnb, patchJSON, putGnbOperation); err != nil {
		logger.WebUILog.Errorw("failed to PUT gNB", "name", gnbName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT gNB"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func postGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	gnbDataBson := configmodels.ToBsonM(gnb)
	return dbadapter.CommonDBClient.RestfulAPIPostManyWithContext(sc, configmodels.GnbDataColl, filter, []interface{}{gnbDataBson})
}

func putGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	gnbDataBson := configmodels.ToBsonM(gnb)
	_, err := dbadapter.CommonDBClient.RestfulAPIPutOneWithContext(sc, configmodels.GnbDataColl, filter, gnbDataBson)
	return err
}

func deleteGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	return dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, configmodels.GnbDataColl, filter)
}

func executeGnbTransaction(ctx context.Context, gnb configmodels.Gnb, patchJSON []byte, operation func(mongo.SessionContext, configmodels.Gnb) error) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err := operation(sc, gnb); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return err
		}
		err = editGnbInNetworkSlices(sc, gnb)
		if err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

func editGnbInNetworkSlices(context context.Context, gnb configmodels.Gnb) error {
	filterByGnb := bson.M{
		"site-info.gNodeBs": bson.M{
			"$elemMatch": bson.M{"name": gnb.Name},
		},
	}
	tacNum, _ := strconv.ParseInt(gnb.Tac, 10, 32)
	update := bson.M{
		"$addToSet": bson.M{
			"site-info.gNodeBs": bson.M{
				"name": gnb.Name,
				"tac":  int32(tacNum),
			},
		},
	}
	_, err := dbadapter.CommonDBClient.GetCollection(sliceDataColl).UpdateMany(context, filterByGnb, update)
	if err != nil {
		return err
	}
	return nil
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
		if err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, configmodels.GnbDataColl, filter); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to delete gNB from collection: %w", err)
		}
		if err = deleteGnbFromNetworkSlices(gnbName, sc); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

func deleteGnbFromNetworkSlices(gnbName string, context context.Context) error {
	filterByGnb := bson.M{
		"site-info.gNodeBs": bson.M{
			"$elemMatch": bson.M{"name": gnbName},
		},
	}
	update := bson.M{
		"site-info.gNodeBs": bson.M{
			"name": gnbName,
		},
	}
	_, err := dbadapter.CommonDBClient.GetCollection(sliceDataColl).UpdateMany(context, filterByGnb, bson.M{"$pull": update})
	if err != nil {
		return err
	}
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
	rawUpfs, err := dbadapter.CommonDBClient.RestfulAPIGetMany(configmodels.UpfDataColl, bson.M{})
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
// @Param        upf  body  configmodels.PostUpfRequest  true  "Hostname and port of the UPF to create"
// @Security     BearerAuth
// @Success      201  {object}  nil  "UPF successfully created"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Error creating UPF"
// @Router       /config/v1/inventory/upf/  [post]
func PostUpf(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a POST UPF request")
	var postUpfParams configmodels.PostUpfRequest
	err := c.ShouldBindJSON(&postUpfParams)
	if err != nil {
		logger.WebUILog.Errorw("invalid UPF POST input parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if !isValidFQDN(postUpfParams.Hostname) {
		errorMessage := fmt.Sprintf("invalid UPF hostname '%s'. Hostname needs to represent a valid FQDN", postUpfParams.Hostname)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	if !isValidUpfPort(postUpfParams.Port) {
		errorMessage := fmt.Sprintf("invalid UPF port '%s'. Port must be a numeric string within the range [0, 65535]", postUpfParams.Port)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	upf := configmodels.Upf(postUpfParams)
	patchJSON, err := getEditUpfPatchJSON(upf)
	if err != nil {
		logger.WebUILog.Errorw("failed to serialize UPF", "hostname", upf.Hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT UPF"})
		return
	}
	if err = executeUpfTransaction(c.Request.Context(), upf, patchJSON, postUpfOperation); err != nil {
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
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a PUT UPF request")
	hostname, _ := c.Params.Get("upf-hostname")
	if !isValidFQDN(hostname) {
		errorMessage := fmt.Sprintf("invalid UPF hostname '%s'. Hostname needs to represent a valid FQDN", hostname)
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
	if !isValidUpfPort(putUpfParams.Port) {
		errorMessage := fmt.Sprintf("invalid UPF port '%s'. Port must be a numeric string within the range [0, 65535]", putUpfParams.Port)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	putUpf := configmodels.Upf{
		Hostname: hostname,
		Port:     putUpfParams.Port,
	}
	patchJSON, err := getEditUpfPatchJSON(putUpf)
	if err != nil {
		logger.WebUILog.Errorw("failed to serialize UPF", "hostname", hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT UPF"})
		return
	}
	if err := executeUpfTransaction(c.Request.Context(), putUpf, patchJSON, putUpfOperation); err != nil {
		logger.WebUILog.Errorw("failed to PUT UPF", "hostname", hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT UPF"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func getEditUpfPatchJSON(upf configmodels.Upf) ([]byte, error) {
	patch := []dbadapter.PatchOperation{
		{
			Op:   "replace",
			Path: "/site-info/upf",
			Value: map[string]string{
				"upf-name": upf.Hostname,
				"upf-port": upf.Port,
			},
		},
	}
	return json.Marshal(patch)
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
	upf := configmodels.Upf{
		Hostname: hostname,
	}
	patch := []dbadapter.PatchOperation{
		{
			Op:   "remove",
			Path: "/site-info/upf",
		},
	}
	patchJSON, _ := json.Marshal(patch)
	if err := executeUpfTransaction(c.Request.Context(), upf, patchJSON, deleteUpfOperation); err != nil {
		logger.WebUILog.Errorw("failed to delete UPF", "hostname", hostname, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed DELETE UPF request for hostname: %v", hostname)
	c.JSON(http.StatusOK, gin.H{})
}
func postUpfOperation(sc mongo.SessionContext, upf configmodels.Upf) error {
	filter := bson.M{"hostname": upf.Hostname}
	upfDataBson := configmodels.ToBsonM(upf)
	if upfDataBson == nil {
		return fmt.Errorf("failed to serialize UPF")
	}
	return dbadapter.CommonDBClient.RestfulAPIPostManyWithContext(sc, configmodels.UpfDataColl, filter, []interface{}{upfDataBson})
}

func putUpfOperation(sc mongo.SessionContext, upf configmodels.Upf) error {
	filter := bson.M{"hostname": upf.Hostname}
	upfDataBson := configmodels.ToBsonM(upf)
	if upfDataBson == nil {
		return fmt.Errorf("failed to serialize UPF")
	}
	_, err := dbadapter.CommonDBClient.RestfulAPIPutOneWithContext(sc, configmodels.UpfDataColl, filter, upfDataBson)
	return err
}

func deleteUpfOperation(sc mongo.SessionContext, upf configmodels.Upf) error {
	filter := bson.M{"hostname": upf.Hostname}
	return dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, configmodels.UpfDataColl, filter)
}

func executeUpfTransaction(ctx context.Context, upf configmodels.Upf, patchJSON []byte, operation func(mongo.SessionContext, configmodels.Upf) error) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err := operation(sc, upf); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return err
		}
		err = updateUpfInNetworkSlices(sc, upf.Hostname, patchJSON)
		if err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorw("failed to abort transaction", "error", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
}

func updateUpfInNetworkSlices(context context.Context, hostname string, patchJSON []byte) error {
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
