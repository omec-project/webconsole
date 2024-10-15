// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

const (
	errorCreateUserAccount    = "failed to create user account"
	errorDeleteAdminAccount   = "deleting an Admin account is not allowed"
	errorDeleteUserAccount    = "failed to delete user account"
	errorIncorrectCredentials = "incorrect username or password. Try again"
	errorInvalidDataProvided  = "invalid data provided"
	errorInvalidPassword      = "Password must have 8 or more characters, must include at least one capital letter, one lowercase letter, and either a number or a symbol."
	errorLogin                = "failed to log in"
	errorMissingPassword      = "password is required"
	errorMissingUsername      = "username is required"
	errorRetrieveUserAccount  = "failed to retrieve user account from DB"
	errorRetrieveUserAccounts = "failed to retrieve user accounts from DB"
	errorUpdateUserAccount    = "failed to update user account"
	errorUsernameNotFound     = "username not found"
	UserAccountDataColl       = "webconsoleData.snapshots.userAccountData"
)

func GetUserAccounts(c *gin.Context) {
	dbUsers, err := fetchDBUsers()
	userResponses := configmodels.TransformDBUsersToUserResponses(dbUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, userResponses)
}

func fetchDBUsers() ([]*configmodels.DBUser, error) {
	rawUsers, err := dbadapter.WebuiDBClient.RestfulAPIGetMany(UserAccountDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, errors.New(errorRetrieveUserAccounts)
	}
	var users []*configmodels.DBUser
	users = make([]*configmodels.DBUser, 0)
	for _, rawUser := range rawUsers {
		var user configmodels.DBUser
		err := json.Unmarshal(configmodels.MapToByte(rawUser), &user)
		if err != nil {
			logger.DbLog.Errorf(errorRetrieveUserAccount)
			continue
		}
		users = append(users, &user)
	}
	return users, nil
}

func GetUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("get user account")
	username := c.Param("username")
	dbUser, err := fetchDBUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	userResponse := configmodels.TransformDBUserToUserResponse(*dbUser)
	c.JSON(http.StatusOK, userResponse)
}

func fetchDBUser(username string) (*configmodels.DBUser, error) {
	filter := bson.M{"username": username}
	rawUser, err := dbadapter.WebuiDBClient.RestfulAPIGetOne(UserAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, err
	}
	if len(rawUser) == 0 {
		return nil, nil
	}
	var user configmodels.DBUser
	err = json.Unmarshal(configmodels.MapToByte(rawUser), &user)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		return nil, err
	}
	return &user, nil
}

func PostUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("create user account")
	var createUserParam configmodels.CreateUserParams
	err := c.ShouldBindJSON(&createUserParam)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if createUserParam.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
		return
	}
	if createUserParam.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(createUserParam.Password) {
		logger.AuthLog.Errorln("invalid password provided")
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	createUserParam.Role = UserRole
	isFirstAccountIssued, err := IsFirstAccountIssued()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	if !isFirstAccountIssued {
		createUserParam.Role = AdminRole
	}
	dbUser, err := configmodels.TransformCreateUserParamsToDBUser(createUserParam)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
	}

	filter := bson.M{"username": dbUser.Username}
	err = dbadapter.WebuiDBClient.RestfulAPIPostMany(UserAccountDataColl, filter, []interface{}{configmodels.ToBsonM(dbUser)})
	if err != nil {
		if strings.Contains(err.Error(), "E11000") {
			logger.DbLog.Errorln("Duplicate username found:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "user account already exists"})
			return
		}
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

func DeleteUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("delete user account")
	username := c.Param("username")
	dbUser, err := fetchDBUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	if dbUser.Role == AdminRole {
		logger.AuthLog.Errorln(errorDeleteAdminAccount)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorDeleteAdminAccount})
		return
	}
	filter := bson.M{"username": username}
	err = dbadapter.WebuiDBClient.RestfulAPIDeleteOne(UserAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorDeleteUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func ChangeUserAccountPasssword(c *gin.Context) {
	logger.WebUILog.Infoln("change user password")
	username := c.Param("username")
	var userParams configmodels.CreateUserParams
	err := c.ShouldBindJSON(&userParams)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if userParams.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(userParams.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	dbUser, err := fetchDBUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	newPasswordDbUser, err := configmodels.TransformCreateUserParamsToDBUser(userParams)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	dbUser.HashedPassword = newPasswordDbUser.HashedPassword
	filter := bson.M{"username": dbUser.Username}
	_, err = dbadapter.WebuiDBClient.RestfulAPIPost(UserAccountDataColl, filter, configmodels.ToBsonM(dbUser))
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func Login(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputUser configmodels.CreateUserParams
		err := c.ShouldBindJSON(&inputUser)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
			return
		}
		if inputUser.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
			return
		}
		if inputUser.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
			return
		}
		dbUser, err := fetchDBUser(inputUser.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if dbUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		if err = bcrypt.CompareHashAndPassword([]byte(dbUser.HashedPassword), []byte(inputUser.Password)); err != nil {
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
		c.JSON(http.StatusOK, gin.H{"token": jwt})
	}
}

func IsFirstAccountIssued() (bool, error) {
	users, err := fetchDBUsers()
	if err != nil {
		return false, err
	}
	return len(users) > 0, nil
}

func validatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasCapital := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumberOrSymbol := regexp.MustCompile(`[0-9!@#$%^&*()_+\-=\[\]{};':"|,.<>?~]`).MatchString(password)
	return hasCapital && hasLower && hasNumberOrSymbol
}
