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
	c.JSON(http.StatusOK, n.inMemoryConfig.accessAndMobility)
	logger.NfConfigLog.Infoln("Handled GET request for access-mobility config")
}

func (n *NFConfigServer) GetPlmnConfig(c *gin.Context) {
	c.JSON(http.StatusOK, n.inMemoryConfig.plmn)
	logger.NfConfigLog.Infoln("Handled GET request for plmn config")
}

func (n *NFConfigServer) GetPlmnSnssaiConfig(c *gin.Context) {
	c.JSON(http.StatusOK, n.inMemoryConfig.plmnSnssai)
	logger.NfConfigLog.Infoln("Handled GET request for plmn-snssai config")
}

func (n *NFConfigServer) GetPolicyControlConfig(c *gin.Context) {
	c.JSON(http.StatusOK, n.inMemoryConfig.policyControl)
	logger.NfConfigLog.Infoln("Handled GET request for policy-control config")
}

func (n *NFConfigServer) GetSessionManagementConfig(c *gin.Context) {
	c.JSON(http.StatusOK, n.inMemoryConfig.sessionManagement)
	logger.NfConfigLog.Infoln("Handled GET request for session-management config")
}
