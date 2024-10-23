// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type StatusResponse struct {
	Initialized bool `json:"initialized"`
}

// GetStatus godoc
//
//	@Description	Get Status. Only available if enableAuthentication is enabled.
//	@Tags			Auth
//	@Success		200	{object}	StatusResponse	"Webui status"
//	@Failure		404	{object}	nil				"Page not found if enableAuthentication is disabled"
//	@Failure		500	{object}	nil				"Internal server error"
//	@Router			/status	[get]
func GetStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		numOfUserAccounts, err := dbadapter.WebuiDBClient.RestfulAPICount(configmodels.UserAccountDataColl, bson.M{})
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't generate status"})
		}
		statusResponse := StatusResponse{
			Initialized: numOfUserAccounts > 0,
		}
		c.JSON(http.StatusOK, statusResponse)
	}
}
