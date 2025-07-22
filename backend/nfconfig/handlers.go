// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/backend/logger"
)

func (n *NFConfigServer) GetAccessMobilityConfig(c *gin.Context) {
	logger.NfConfigLog.Debugf("Handling GET request for access-mobility config %+v", n.inMemoryConfig.accessAndMobility)
	c.JSON(http.StatusOK, n.inMemoryConfig.accessAndMobility)
}

func (n *NFConfigServer) GetPlmnConfig(c *gin.Context) {
	logger.NfConfigLog.Debugf("Handling GET request for plmn config %+v", n.inMemoryConfig.plmn)
	c.JSON(http.StatusOK, n.inMemoryConfig.plmn)
}

func (n *NFConfigServer) GetPlmnSnssaiConfig(c *gin.Context) {
	logger.NfConfigLog.Debugf("Handling GET request for plmn-snssai config %+v", n.inMemoryConfig.plmnSnssai)
	c.JSON(http.StatusOK, n.inMemoryConfig.plmnSnssai)
}

func (n *NFConfigServer) GetPolicyControlConfig(c *gin.Context) {
	logger.NfConfigLog.Debugf("Handling GET request for policy-control config %+v", n.inMemoryConfig.policyControl)
	c.JSON(http.StatusOK, n.inMemoryConfig.policyControl)
}

func (n *NFConfigServer) GetSessionManagementConfig(c *gin.Context) {
	logger.NfConfigLog.Debugf("Handling GET request for session-management config %+v", n.inMemoryConfig.sessionManagement)
	c.JSON(http.StatusOK, n.inMemoryConfig.sessionManagement)
}

func (n *NFConfigServer) GetImsiQosConfig(c *gin.Context) {
	dnn := c.Param("dnn")
	imsi := strings.TrimPrefix(c.Param("imsi"), "imis-")
	logger.NfConfigLog.Debugf("Handling GET request for QoS config for IMSI %s", imsi)
	imsiQos := []nfConfigApi.ImsiQos{}
	for _, imsiQosConfig := range n.inMemoryConfig.imsiQos {
		if imsiQosConfig.dnn == dnn && slices.Contains(imsiQosConfig.imsis, imsi) {
			imsiQos = imsiQosConfig.qos
			break
		}
	}
	c.JSON(http.StatusOK, imsiQos)
}
