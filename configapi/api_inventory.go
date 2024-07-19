// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
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

//var configChannel chan *configmodels.ConfigMessage

//var configLog *logrus.Entry

func GetGnbs(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Get all gNBs")

	rawGnbs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(gnbDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Warnln(errGetMany)
	}

	var gnbs []*configmodels.Gnb
	gnbs = make([]*configmodels.Gnb, 0)
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
	logger.ConfigLog.Debugf("Received GnbPostByName ")
	if ret := gnbPostHandler(c); ret == true {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func GnbDeleteByName(c *gin.Context) {
	logger.ConfigLog.Debugf("Received GnbDeleteByName ")
	if ret := gnbDeletetHandler(c); ret == true {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func gnbPostHandler(c *gin.Context) bool {
	var gnbName string
	var exists bool
	if gnbName, exists = c.Params.Get("gnb-name"); !exists {
		configLog.Infof("Post gNB request is missing gnb-name")
		return false
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
		configLog.Infof(" err %v", err)
		return false
	}
	req := httpwrapper.NewRequest(c.Request, request)
	procReq := req.Body.(configmodels.Gnb)

	var msg configmodels.ConfigMessage
	procReq.GnbName = gnbName
	msg.MsgType = configmodels.Inventory
	msg.MsgMethod = configmodels.Post_op
	msg.Gnb = &procReq
	msg.GnbName = gnbName
	configChannel <- &msg
	configLog.Infof("Successfully added gNB [%v] to config channel.", gnbName)
	return true
}

func gnbDeletetHandler(c *gin.Context) bool {
	var gnbName string
	var exists bool
	if gnbName, exists = c.Params.Get("gnb-name"); !exists {
		configLog.Infof("Delete gNB request is missing gnb-name")
		return false
	}
	configLog.Infof("Received delete gNB %v request", gnbName)
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Inventory
	msg.MsgMethod = configmodels.Delete_op
	msg.GnbName = gnbName
	configChannel <- &msg
	configLog.Infof("Successfully added gNB [%v] with delete_op to config channel.", gnbName)
	return true
}

func GetUpfs(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Get all UPFs")

	rawUpfs, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(upfDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Warnln(errGetMany)
	}

	var upfs []*configmodels.Upf
	upfs = make([]*configmodels.Upf, 0)
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
	logger.ConfigLog.Debugf("Received UpfPostByName ")
	if ret := upfPostHandler(c); ret == true {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func UpfDeleteByName(c *gin.Context) {
	logger.ConfigLog.Debugf("Received UpfDeleteByName ")
	if ret := upfDeletetHandler(c); ret == true {
		c.JSON(http.StatusOK, gin.H{})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
}

func upfPostHandler(c *gin.Context) bool {
	var upfHostname string
	var exists bool
	if upfHostname, exists = c.Params.Get("upf-hostname"); !exists {
		configLog.Infof("Post UPF request is missing upf-hostname")
		return false
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
		configLog.Infof(" err %v", err)
		return false
	}
	req := httpwrapper.NewRequest(c.Request, request)
	procReq := req.Body.(configmodels.Upf)

	var msg configmodels.ConfigMessage
	procReq.Hostname = upfHostname
	msg.MsgType = configmodels.Inventory
	msg.MsgMethod = configmodels.Post_op
	msg.Upf = &procReq
	msg.UpfHostname = upfHostname
	configChannel <- &msg
	configLog.Infof("Successfully added UPF [%v] to config channel.", upfHostname)
	return true
}

func upfDeletetHandler(c *gin.Context) bool {
	var upfHostname string
	var exists bool
	if upfHostname, exists = c.Params.Get("upf-hostname"); !exists {
		configLog.Infof("Delete UPF request is missing upf-hostname")
		return false
	}
	configLog.Infof("Received Delete UPF %v", upfHostname)
	var msg configmodels.ConfigMessage
	msg.MsgType = configmodels.Inventory
	msg.MsgMethod = configmodels.Delete_op
	msg.UpfHostname = upfHostname
	configChannel <- &msg
	configLog.Infof("Successfully added UPF [%v] with delete_op to config channel.", upfHostname)
	return true
}
