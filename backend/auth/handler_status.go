// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
)

type StatusResponse struct {
	Initialized bool `json:"initialized"`
}

func GetStatus(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {

		firstAccountIssued, err := IsFirstAccountIssued()
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't generate status"})
		}
		statusResponse := StatusResponse{
			Initialized: firstAccountIssued,
		}
		c.JSON(http.StatusOK, statusResponse)
	}

}
