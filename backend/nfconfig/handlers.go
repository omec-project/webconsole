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

func (n *NFConfig) GetAccessMobilityConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for access-mobility config")
	c.JSON(http.StatusOK, []AccessMobilityConfig{})
}

func (n *NFConfig) GetPlmnConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for plmn config")
	c.JSON(http.StatusOK, []PlmnConfig{})
}

func (n *NFConfig) GetPlmnSnssaiConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for plmn-snssai config")
	c.JSON(http.StatusOK, []PlmnSnssaiConfig{})
}

func (n *NFConfig) GetPolicyControlConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for policy-control config")
	c.JSON(http.StatusOK, []PolicyControlConfig{})
}

func (n *NFConfig) GetSessionManagementConfig(c *gin.Context) {
	logger.ConfigLog.Infoln("Handling GET request for session-management config")
	c.JSON(http.StatusOK, []SessionManagementConfig{})
}
