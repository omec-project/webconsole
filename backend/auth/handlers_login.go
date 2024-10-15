// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/backend/logger"
	"golang.org/x/crypto/bcrypt"
)

const errorLogin = "failed to log in"

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func Login(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginParams LoginParams
		err := c.ShouldBindJSON(&loginParams)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
			return
		}
		if loginParams.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
			return
		}
		if loginParams.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
			return
		}
		dbUser, err := fetchDBUserAccount(loginParams.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if dbUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		if err = bcrypt.CompareHashAndPassword([]byte(dbUser.HashedPassword), []byte(loginParams.Password)); err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		jwt, err := generateJWT(dbUser.Username, dbUser.Role, jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorLogin})
			return
		}
		loginResponse := LoginResponse{
			Token: jwt,
		}
		c.JSON(http.StatusOK, loginResponse)
	}
}

func expireAfter() int64 {
	return time.Now().Add(time.Hour * 1).Unix()
}

func generateJWT(username string, role int, jwtSecret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtGocertClaims{
		Username: username,
		Role:     role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireAfter(),
		},
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
