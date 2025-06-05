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
	logger.ConfigLog.Infoln("Handling GET request for access-mobility config")
	c.JSON(http.StatusOK, []AccessAndMobilityConfig{})
}

func (n *NFConfigServer) GetPlmnConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for plmn config")
	c.JSON(http.StatusOK, []PlmnConfig{})
}

func (n *NFConfigServer) GetPlmnSnssaiConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for plmn-snssai config")
	c.JSON(http.StatusOK, []PlmnSnssaiConfig{})
}

func (n *NFConfigServer) GetPolicyControlConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for policy-control config")
	c.JSON(http.StatusOK, []PolicyControlConfig{})
}

func (n *NFConfigServer) GetSessionManagementConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for session-management config")
	c.JSON(http.StatusOK, []SessionManagementConfig{})
}
