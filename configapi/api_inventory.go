// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"errors"
	"fmt"
	"encoding/json"
	"strings"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/util/httpwrapper"
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

func GetGnbs(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("Get all gNBs")

	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
	rawGnbs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(gnbDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorln(errGetMany)
		c.JSON(http.StatusInternalServerError, gnbs)
	}

	for _, rawGnb := range rawGnbs {
		var gnbData configmodels.Gnb
		err := json.Unmarshal(mapToByte(rawGnb), &gnbData)
		if err != nil {
			logger.DbLog.Errorf("Could not unmarshall gNB %v", rawGnb)
		}
		gnbs = append(gnbs, &gnbData)
	}
	c.JSON(http.StatusOK, gnbs)
}

func GnbPostByName(c *gin.Context) {
	setInventoryCorsHeader(c)
	if ret := gnbPostHandler(c); ret == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func GnbDeleteByName(c *gin.Context) {
	setInventoryCorsHeader(c)
	if ret := gnbDeletetHandler(c); ret == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func gnbPostHandler(c *gin.Context) error {
	var gnbName string
	var exists bool
	if gnbName, exists = c.Params.Get("gnb-name"); !exists {
		errorMessage := "Post gNB request is missing gnb-name"
		configLog.Errorf(errorMessage)
		return errors.New(errorMessage)
	}
	configLog.Infof("Received gNB %v", gnbName)
	var err error
	var request configmodels.Gnb

	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		configLog.Errorf("err %v", err)
		return fmt.Errorf("Failed to create gNB %v: %w", gnbName, err)
	}
	req := httpwrapper.NewRequest(c.Request, request)
	procReq := req.Body.(configmodels.Gnb)
	procReq.GnbName = gnbName
	var msg = configmodels.ConfigMessage{
		MsgType:     configmodels.Inventory,
		MsgMethod:   configmodels.Post_op,
		GnbName: gnbName,
		Gnb: &procReq,
	}
	configChannel <- &msg
	configLog.Infof("Successfully added gNB [%v] to config channel.", gnbName)
	return nil
}

func gnbDeletetHandler(c *gin.Context) error {
	var gnbName string
	var exists bool
	if gnbName, exists = c.Params.Get("gnb-name"); !exists {
		errorMessage := "Delete gNB request is missing gnb-name"
		configLog.Errorf(errorMessage)
		return fmt.Errorf(errorMessage)
	}
	configLog.Infof("Received delete gNB %v request", gnbName)
	var msg = configmodels.ConfigMessage{
		MsgType:   configmodels.Inventory,
		MsgMethod: configmodels.Delete_op,
		GnbName:   gnbName,
	}
	configChannel <- &msg
	configLog.Infof("Successfully added gNB [%v] with delete_op to config channel.", gnbName)
	return nil
}

func GetUpfs(c *gin.Context) {
	setInventoryCorsHeader(c)
	logger.WebUILog.Infoln("Get all UPFs")

	var upfs []*configmodels.Upf
	upfs = make([]*configmodels.Upf, 0)
	rawUpfs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(upfDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorln(errGetMany)
		c.JSON(http.StatusInternalServerError, upfs)
	}

	for _, rawUpf := range rawUpfs {
		var upfData configmodels.Upf
		err := json.Unmarshal(mapToByte(rawUpf), &upfData)
		if err != nil {
			logger.DbLog.Errorf("Could not unmarshall UPF %v", rawUpf)
		}
		upfs = append(upfs, &upfData)
	}
	c.JSON(http.StatusOK, upfs)
}

func UpfPostByName(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := upfPostHandler(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func UpfDeleteByName(c *gin.Context) {
	setInventoryCorsHeader(c)
	if err := upfDeleteHandler(c); err == nil {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func upfPostHandler(c *gin.Context) error {
	var upfHostname string
	var exists bool
	if upfHostname, exists = c.Params.Get("upf-hostname"); !exists {
		errorMessage := "Post UPF request is missing upf-hostname"
		configLog.Errorf(errorMessage)
		return errors.New(errorMessage)
	}
	configLog.Infof("Received UPF %v", upfHostname)
	var err error
	var request configmodels.Upf

	s := strings.Split(c.GetHeader("Content-Type"), ";")
	switch s[0] {
	case "application/json":
		err = c.ShouldBindJSON(&request)
	}
	if err != nil {
		configLog.Errorf("err %v", err)
		return fmt.Errorf("Failed to create UPF %v: %w", upfHostname, err)
	}
	req := httpwrapper.NewRequest(c.Request, request)
	procReq := req.Body.(configmodels.Upf)
	procReq.Hostname = upfHostname
	var msg = configmodels.ConfigMessage{
		MsgType:     configmodels.Inventory,
		MsgMethod:   configmodels.Post_op,
		UpfHostname: upfHostname,
		Upf: &procReq,
	}
	configChannel <- &msg
	configLog.Infof("Successfully added UPF [%v] to config channel.", upfHostname)
	return nil
}

func upfDeleteHandler(c *gin.Context) error {
	var upfHostname string
	var exists bool
	if upfHostname, exists = c.Params.Get("upf-hostname"); !exists {
		errorMessage := "Delete UPF request is missing upf-hostname"
		configLog.Errorf(errorMessage)
		return fmt.Errorf(errorMessage)
	}
	configLog.Infof("Received Delete UPF %v", upfHostname)
	var msg = configmodels.ConfigMessage{
		MsgType:     configmodels.Inventory,
		MsgMethod:   configmodels.Delete_op,
		UpfHostname: upfHostname,
	}
	configChannel <- &msg
	configLog.Infof("Successfully added UPF [%v] with delete_op to config channel.", upfHostname)
	return nil
}
