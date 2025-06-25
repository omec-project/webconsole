// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

func (n *NFConfigServer) GetAccessMobilityConfig(c *gin.Context) {
	logger.NfConfigLog.Infoln("Handling GET request for access-mobility config")
	c.JSON(http.StatusOK, n.inMemoryConfig.accessAndMobility)
}

func (n *NFConfigServer) GetPlmnConfig(c *gin.Context) {
	logger.NfConfigLog.Infoln("Handling GET request for plmn config")
	c.JSON(http.StatusOK, n.inMemoryConfig.plmn)
}

func (n *NFConfigServer) GetPlmnSnssaiConfig(c *gin.Context) {
	logger.NfConfigLog.Infoln("Handling GET request for plmn-snssai config")
	c.JSON(http.StatusOK, n.inMemoryConfig.plmnSnssai)
}

func (n *NFConfigServer) GetPolicyControlConfig(c *gin.Context) {
	logger.NfConfigLog.Infoln("Handling GET request for policy-control config")
	c.JSON(http.StatusOK, n.inMemoryConfig.policyControl)
}

func (n *NFConfigServer) GetSessionManagementConfig(c *gin.Context) {
	logger.NfConfigLog.Infoln("Handling GET request for session-management config")
	c.JSON(http.StatusOK, n.inMemoryConfig.sessionManagement)
}
