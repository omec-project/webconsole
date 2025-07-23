// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
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
		logger.DbLog.Errorf("failed to retrieve gNBs with error: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve gNBs"})
		return
	}

	for _, rawGnb := range rawGnbs {
		var gnbData configmodels.Gnb
		err = json.Unmarshal(configmodels.MapToByte(rawGnb), &gnbData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshal gNB %s", rawGnb)
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
// @Success     201  {object}  nil  "gNB successfully created"
// @Failure     409  {object}  nil  "Resource Conflict"
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
		logger.WebUILog.Errorf("invalid UPF gNB input parameters with error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if !isValidName(postGnbParams.Name) {
		errorMessage := fmt.Sprintf("invalid gNB name '%s'. Name needs to match the following regular expression: %s", postGnbParams.Name, NAME_PATTERN)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	if postGnbParams.Tac != nil {
		if !isValidGnbTac(*postGnbParams.Tac) {
			errorMessage := fmt.Sprintf("invalid gNB TAC '%+v'. TAC must be an integer within the range [1, 16777215]", *postGnbParams.Tac)
			logger.WebUILog.Errorln(errorMessage)
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
			return
		}
	}
	gnb := configmodels.Gnb(postGnbParams)
	if err := executeGnbTransaction(c.Request.Context(), gnb, updateGnbInNetworkSlices, postGnbOperation); err != nil {
		if strings.Contains(err.Error(), "E11000") {
			logger.WebUILog.Errorf("duplicate gNB name found error: %+v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "gNB already exists"})
			return
		}
		logger.WebUILog.Errorf("failed to create gNB with name: %s with error: %+v", postGnbParams.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create gNB"})
		return
	}
	logger.WebUILog.Infof("successfully executed POST gNB %s request", postGnbParams.Name)
	c.JSON(http.StatusCreated, gin.H{})
}

func postGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	gnbDataBson := configmodels.ToBsonM(gnb)
	return dbadapter.CommonDBClient.RestfulAPIPostManyWithContext(sc, configmodels.GnbDataColl, filter, []interface{}{gnbDataBson})
}

// PutGnb godoc
//
// @Description Create or update a gNB
// @Tags        gNBs
// @Produce     json
// @Param       gnb-name    path    string                        true    "Name of the gNB"
// @Param       tac         body    configmodels.PutGnbRequest    true    "TAC of the gNB"
// @Security    BearerAuth
// @Success     201  {object}  nil  "gNB successfully created"
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
		logger.WebUILog.Errorf("invalid gNB PUT input parameters for gnbname: %s error: %+v", gnbName, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if !isValidGnbTac(putGnbParams.Tac) {
		errorMessage := fmt.Sprintf("invalid gNB TAC '%+v'. TAC must be an integer within the range [1, 16777215]", putGnbParams.Tac)
		logger.WebUILog.Errorln(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return
	}
	putGnb := configmodels.Gnb{
		Name: gnbName,
		Tac:  &putGnbParams.Tac,
	}
	if err := executeGnbTransaction(c.Request.Context(), putGnb, updateGnbInNetworkSlices, putGnbOperation); err != nil {
		logger.WebUILog.Errorf("failed to PUT gNB name: %s error: %+v", gnbName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT gNB"})
		return
	}
	logger.WebUILog.Infof("successfully executed PUT gNB request for hostname: %s", gnbName)
	c.JSON(http.StatusOK, gin.H{})
}

func putGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	gnbDataBson := configmodels.ToBsonM(gnb)
	_, err := dbadapter.CommonDBClient.RestfulAPIPutOneWithContext(sc, configmodels.GnbDataColl, filter, gnbDataBson)
	return err
}

func updateGnbInNetworkSlices(gnb configmodels.Gnb) error {
	filterByGnb := bson.M{
		"site-info.gNodeBs.name": gnb.Name,
	}
	statusCode, err := updateInventoryInNetworkSlices(filterByGnb, func(networkSlice *configmodels.Slice) {
		for i := range networkSlice.SiteInfo.GNodeBs {
			if networkSlice.SiteInfo.GNodeBs[i].Name == gnb.Name {
				networkSlice.SiteInfo.GNodeBs[i].Tac = *gnb.Tac
			}
		}
	})
	if err != nil {
		logger.ConfigLog.Errorf("failed to update gNB in network slices: %+v", err)
	}
	logger.ConfigLog.Infof("update gNB result statusCode: %d", statusCode)
	return err
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
	gnb := configmodels.Gnb{
		Name: gnbName,
	}
	err := executeGnbTransaction(c.Request.Context(), gnb, removeGnbFromNetworkSlices, deleteGnbOperation)
	if err != nil {
		logger.WebUILog.Errorf("failed to delete GNB with name %s error: %+v", gnbName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete gNB"})
		return
	}
	logger.WebUILog.Infof("successfully executed DELETE gNB %s request", gnbName)
	c.JSON(http.StatusOK, gin.H{})
}

func deleteGnbOperation(sc mongo.SessionContext, gnb configmodels.Gnb) error {
	filter := bson.M{"name": gnb.Name}
	return dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, configmodels.GnbDataColl, filter)
}

func removeGnbFromNetworkSlices(gnb configmodels.Gnb) error {
	filterByGnb := bson.M{
		"site-info.gNodeBs.name": gnb.Name,
	}
	statusCode, err := updateInventoryInNetworkSlices(filterByGnb, func(networkSlice *configmodels.Slice) {
		networkSlice.SiteInfo.GNodeBs = slices.DeleteFunc(networkSlice.SiteInfo.GNodeBs, func(existingGnb configmodels.SliceSiteInfoGNodeBs) bool {
			return gnb.Name == existingGnb.Name
		})
	})
	if err != nil {
		logger.ConfigLog.Errorf("failed to remove gNB from network slices: %+v", err)
	}
	logger.ConfigLog.Infof("remove gNB result statusCode: %d", statusCode)
	return err
}

func executeGnbTransaction(ctx context.Context, gnb configmodels.Gnb, nsOperation func(configmodels.Gnb) error, gnbOperation func(mongo.SessionContext, configmodels.Gnb) error) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err = session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err = gnbOperation(sc, gnb); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorf("failed to abort transaction with error: %+v", abortErr)
			}
			return err
		}
		err = nsOperation(gnb)
		if err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorf("failed to abort transaction with error: %+v", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %w", err)
		}
		return session.CommitTransaction(sc)
	})
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
		logger.DbLog.Errorf("failed to retrieve UPFs with error: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve UPFs"})
		return
	}

	for _, rawUpf := range rawUpfs {
		var upfData configmodels.Upf
		err := json.Unmarshal(configmodels.MapToByte(rawUpf), &upfData)
		if err != nil {
			logger.DbLog.Errorf("could not unmarshal UPF %s", rawUpf)
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
// @Failure      409  {object}  nil  "Resource Conflict"
// @Failure      500  {object}  nil  "Error creating UPF"
// @Router       /config/v1/inventory/upf/  [post]
func PostUpf(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("received a POST UPF request")
	var postUpfParams configmodels.PostUpfRequest
	err := c.ShouldBindJSON(&postUpfParams)
	if err != nil {
		logger.WebUILog.Errorf("invalid UPF POST input parameters error: %v+", err)
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
	if err = executeUpfTransaction(c.Request.Context(), upf, updateUpfInNetworkSlices, postUpfOperation); err != nil {
		if strings.Contains(err.Error(), "E11000") {
			logger.WebUILog.Errorf("duplicate hostname found with error: %+v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "UPF already exists"})
			return
		}
		logger.WebUILog.Errorf("failed to create UPF with hostname: %s with error: %+v", postUpfParams.Hostname, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed POST UPF %s request", postUpfParams.Hostname)
	c.JSON(http.StatusCreated, gin.H{})
}

func postUpfOperation(sc mongo.SessionContext, upf configmodels.Upf) error {
	filter := bson.M{"hostname": upf.Hostname}
	upfDataBson := configmodels.ToBsonM(upf)
	if upfDataBson == nil {
		return fmt.Errorf("failed to serialize UPF")
	}
	return dbadapter.CommonDBClient.RestfulAPIPostManyWithContext(sc, configmodels.UpfDataColl, filter, []interface{}{upfDataBson})
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
		logger.WebUILog.Errorf("invalid UPF PUT input parameters with hostname: %s with error: %+v", hostname, err)
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
	if err := executeUpfTransaction(c.Request.Context(), putUpf, updateUpfInNetworkSlices, putUpfOperation); err != nil {
		logger.WebUILog.Errorf("failed to PUT UPF with hostname: %s with error: %+v", hostname, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to PUT UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed PUT UPF request for hostname: %s", hostname)
	c.JSON(http.StatusOK, gin.H{})
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

func updateUpfInNetworkSlices(upf configmodels.Upf) error {
	filterByUpf := bson.M{"site-info.upf.upf-name": upf.Hostname}
	statusCode, err := updateInventoryInNetworkSlices(filterByUpf, func(networkSlice *configmodels.Slice) {
		networkSlice.SiteInfo.Upf = map[string]interface{}{
			"upf-name": upf.Hostname,
			"upf-port": upf.Port,
		}
	})
	if err != nil {
		logger.ConfigLog.Errorf("failed to update UPF in network slices: %+v", err)
	}
	logger.ConfigLog.Infof("update UPF result statusCode: %d", statusCode)
	return err
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
	if err := executeUpfTransaction(c.Request.Context(), upf, removeUpfFromNetworkSlices, deleteUpfOperation); err != nil {
		logger.WebUILog.Errorf("failed to delete UPF with hostname: %s with error: %+v", hostname, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete UPF"})
		return
	}
	logger.WebUILog.Infof("successfully executed DELETE UPF request for hostname: %s", hostname)
	c.JSON(http.StatusOK, gin.H{})
}

func deleteUpfOperation(sc mongo.SessionContext, upf configmodels.Upf) error {
	filter := bson.M{"hostname": upf.Hostname}
	return dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(sc, configmodels.UpfDataColl, filter)
}

func removeUpfFromNetworkSlices(upf configmodels.Upf) error {
	filterByUpf := bson.M{"site-info.upf.upf-name": upf.Hostname}
	statusCode, err := updateInventoryInNetworkSlices(filterByUpf, func(networkSlice *configmodels.Slice) {
		networkSlice.SiteInfo.Upf = nil
	})
	if err != nil {
		logger.ConfigLog.Errorf("failed to remove UPF from network slices: %+v", err)
	}
	logger.ConfigLog.Infof("remove UPF result statusCode: %d", statusCode)
	return err
}

func executeUpfTransaction(ctx context.Context, upf configmodels.Upf, nsOperation func(configmodels.Upf) error, upfOperation func(mongo.SessionContext, configmodels.Upf) error) error {
	session, err := dbadapter.CommonDBClient.StartSession()
	if err != nil {
		return fmt.Errorf("failed to initialize DB session: %w", err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err = session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		if err = upfOperation(sc, upf); err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorf("failed to abort transaction with error: %+v", abortErr)
			}
			return err
		}
		err = nsOperation(upf)
		if err != nil {
			if abortErr := session.AbortTransaction(sc); abortErr != nil {
				logger.DbLog.Errorf("failed to abort transaction with error: %+v", abortErr)
			}
			return fmt.Errorf("failed to update network slices: %+v", err)
		}
		return session.CommitTransaction(sc)
	})
}

func updateInventoryInNetworkSlices(filter bson.M, updateFunc func(*configmodels.Slice)) (int, error) {
	rawNetworkSlices, err := dbadapter.CommonDBClient.RestfulAPIGetMany(sliceDataColl, filter)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to fetch network slices: %w", err)
	}

	for _, rawNetworkSlice := range rawNetworkSlices {
		var networkSlice configmodels.Slice
		if err = json.Unmarshal(configmodels.MapToByte(rawNetworkSlice), &networkSlice); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error unmarshaling network slice: %w", err)
		}
		prevSlice := getSliceByName(networkSlice.SliceName)
		updateFunc(&networkSlice)
		if statusCode, err := updateNS(networkSlice, *prevSlice); err != nil {
			logger.ConfigLog.Errorf("Error updating slice %s: %+v", networkSlice.SliceName, err)
			return statusCode, err
		}

	}
	return http.StatusOK, nil
}
