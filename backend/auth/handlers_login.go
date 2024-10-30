// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

const (
	errorIncorrectCredentials = "incorrect username or password. Try again"
	errorInvalidDataProvided  = "invalid data provided"
	errorLogin                = "failed to log in"
	errorMissingPassword      = "password is required"
	errorMissingUsername      = "username is required"
	errorRetrieveUserAccount  = "failed to retrieve user account"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// LoginPost godoc
//
// @Description  Log in. Only available if enableAuthentication is enabled.
// @Tags         Auth
// @Param        loginParams    body    LoginParams    true    " "
// @Success      200  {object}  LoginResponse  "Authorization token"
// @Failure      400  {object}  nil            "Bad request"
// @Failure      401  {object}  nil            "Authentication failed"
// @Failure      404  {object}  nil            "Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil            "Internal server error"
// @Router       /login  [post]
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

		filter := bson.M{"username": loginParams.Username}
		rawUserAccount, err := dbadapter.WebuiDBClient.RestfulAPIGetOne(configmodels.UserAccountDataColl, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if len(rawUserAccount) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		var dbUser configmodels.DBUserAccount
		err = json.Unmarshal(configmodels.MapToByte(rawUserAccount), &dbUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if err = bcrypt.CompareHashAndPassword([]byte(dbUser.HashedPassword), []byte(loginParams.Password)); err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		token, err := GenerateJWT(dbUser.Username, dbUser.Role, jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorLogin})
			return
		}
		loginResponse := LoginResponse{
			Token: token,
		}
		c.JSON(http.StatusOK, loginResponse)
	}
}

func expireAfter() int64 {
	return time.Now().Add(time.Hour * 1).Unix()
}

func GenerateJWT(username string, role int, jwtSecret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtWebconsoleClaims{
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
