// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
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
	logger.WebUILog.Infoln("get all gNBs")

	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
	rawGnbs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(gnbDataColl, bson.M{})
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
	rawUpfs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(upfDataColl, bson.M{})
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

func handlePostUpf(c *gin.Context) error {
	upfHostname, exists := c.Params.Get("upf-hostname")
	if !exists {
		errorMessage := "post UPF request is missing upf-hostname"
		logger.ConfigLog.Errorln(errorMessage)
		return fmt.Errorf("%s", errorMessage)
	}
	logger.ConfigLog.Infof("received UPF %v", upfHostname)
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return fmt.Errorf("invalid header")
	}
	var postUpfRequest configmodels.PostUpfRequest
	err := c.ShouldBindJSON(&postUpfRequest)
	if err != nil {
		logger.ConfigLog.Errorf("err %v", err)
		return fmt.Errorf("invalid JSON format")
	}
	if postUpfRequest.Port == "" {
		errorMessage := "post UPF request body is missing port"
		logger.ConfigLog.Errorln(errorMessage)
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
	logger.ConfigLog.Infof("successfully added UPF [%v] to config channel", upfHostname)
	return nil
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
