// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package configapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	errorCreateUserAccount    = "failed to create user account"
	errorDeleteAdminAccount   = "deleting an admin user account is not allowed"
	errorDeleteUserAccount    = "failed to delete user account"
	errorIncorrectCredentials = "incorrect username or password. Try again"
	errorInvalidDataProvided  = "invalid data provided"
	errorInvalidPassword      = "password must have 8 or more characters, must include at least one capital letter, one lowercase letter, and either a number or a symbol."
	errorMissingPassword      = "password is required"
	errorMissingUsername      = "username is required"
	errorRetrieveUserAccount  = "failed to retrieve user account"
	errorRetrieveUserAccounts = "failed to retrieve user accounts"
	errorUpdateUserAccount    = "failed to update user account"
	errorUsernameNotFound     = "username not found"
)

// GetUserAccounts godoc
//
// @Description  Return the list of user accounts
// @Tags         User Accounts
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   configmodels.GetUserAccountResponse  "List of user accounts"
// @Failure      401  {object}  nil                                  "Authorization failed"
// @Failure      403  {object}  nil                                  "Forbidden"
// @Failure      404  {object}  nil                                  "Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil                                  "Error retrieving user accounts"
// @Router       /config/v1/account/  [get]
func GetUserAccounts(c *gin.Context) {
	logger.WebUILog.Infoln("get user accounts")
	rawUsers, err := dbadapter.WebuiDBClient.RestfulAPIGetMany(configmodels.UserAccountDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	userResponses := make([]*configmodels.GetUserAccountResponse, 0, len(rawUsers))
	for _, rawUser := range rawUsers {
		var dbUserAccount configmodels.DBUserAccount
		err := json.Unmarshal(configmodels.MapToByte(rawUser), &dbUserAccount)
		if err != nil {
			logger.DbLog.Errorf(errorRetrieveUserAccount)
			continue
		}
		userResponse := &configmodels.GetUserAccountResponse{
			Username: dbUserAccount.Username,
			Role:     dbUserAccount.Role,
		}
		userResponses = append(userResponses, userResponse)
	}
	c.JSON(http.StatusOK, userResponses)
}

// GetUserAccount godoc
//
// @Description  Return the user account
// @Tags         User Accounts
// @Produce      json
// @Param        username    path    string    true    "Username of the user account"
// @Security     BearerAuth
// @Success      200  {object}  configmodels.GetUserAccountResponse  "User account"
// @Failure      401  {object}  nil                                  "Authorization failed"
// @Failure      403  {object}  nil                                  "Forbidden"
// @Failure      404  {object}  nil                                  "User account not found. Or Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil                                  "Error retrieving user account"
// @Router      /config/v1/account/{username}  [get]
func GetUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("get user account")
	username := c.Param("username")
	dbUserAccount, err := fetchDBUserAccount(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUserAccount == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	userResponse := configmodels.GetUserAccountResponse{
		Username: dbUserAccount.Username,
		Role:     dbUserAccount.Role,
	}
	c.JSON(http.StatusOK, userResponse)
}

func fetchDBUserAccount(username string) (*configmodels.DBUserAccount, error) {
	filter := bson.M{"username": username}
	rawUserAccount, err := dbadapter.WebuiDBClient.RestfulAPIGetOne(configmodels.UserAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, err
	}
	if len(rawUserAccount) == 0 {
		return nil, nil
	}
	var userAccount configmodels.DBUserAccount
	err = json.Unmarshal(configmodels.MapToByte(rawUserAccount), &userAccount)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		return nil, err
	}
	return &userAccount, nil
}

// CreateUserAccount godoc
//
// @Description  Create a new user account
// @Tags         User Accounts
// @Produce      json
// @Param        params    body    configmodels.CreateUserAccountParams    true    "Username and password"
// @Security     BearerAuth
// @Success      200  {object}  nil  "User account created"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      404  {object}  nil  "Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil  "Failed to create the user account"
// @Router      /config/v1/account/  [post]
func CreateUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("create user account")
	var createUserParams configmodels.CreateUserAccountParams
	err := c.ShouldBindJSON(&createUserParams)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if createUserParams.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
		return
	}
	if createUserParams.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(createUserParams.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	newUserRole := configmodels.UserRole
	isFirstAccountIssued, err := isFirstAccountIssued()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	if !isFirstAccountIssued {
		newUserRole = configmodels.AdminRole
	}
	dbUser, err := configmodels.CreateNewDBUserAccount(createUserParams.Username, createUserParams.Password, newUserRole)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
		return
	}

	filter := bson.M{"username": dbUser.Username}
	err = dbadapter.WebuiDBClient.RestfulAPIPostMany(configmodels.UserAccountDataColl, filter, []interface{}{configmodels.ToBsonM(dbUser)})
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

// DeleteUserAccount godoc
//
// @Description  Delete an existing user account
// @Tags         User Accounts
// @Produce      json
// @Param        username    path    string    true    "Username of the user account"
// @Security     BearerAuth
// @Success      200  {object}  nil  "User account deleted"
// @Failure      400  {object}  nil  "Failed to delete the user account"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      404  {object}  nil  "User account not found. Or Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil  "Failed to delete the user account"
// @Router      /config/v1/account/{username}  [delete]
func DeleteUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("delete user account")
	username := c.Param("username")
	dbUserAccount, err := fetchDBUserAccount(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUserAccount == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	if dbUserAccount.Role == configmodels.AdminRole {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorDeleteAdminAccount})
		return
	}
	filter := bson.M{"username": username}
	err = dbadapter.WebuiDBClient.RestfulAPIDeleteOne(configmodels.UserAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorDeleteUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// ChangeUserAccountPasssword godoc
//
// @Description  Create a new user account
// @Tags         User Accounts
// @Produce      json
// @Param        username    path    string                               true    "Username"
// @Param        params      body    configmodels.ChangePasswordParams    true    "Username and password"
// @Security     BearerAuth
// @Success      200  {object}  nil  "Password changed"
// @Failure      400  {object}  nil  "Bad request"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      404  {object}  nil  "Page not found if enableAuthentication is disabled"
// @Failure      500  {object}  nil  "Failed to update the user account"
// @Router      /config/v1/account/{username}/change_password  [post]
func ChangeUserAccountPasssword(c *gin.Context) {
	logger.WebUILog.Infoln("change user password")
	username := c.Param("username")
	var changePasswordParams configmodels.ChangePasswordParams
	err := c.ShouldBindJSON(&changePasswordParams)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if changePasswordParams.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(changePasswordParams.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	dbUser, err := fetchDBUserAccount(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	newPasswordDbUser, err := configmodels.CreateNewDBUserAccount(dbUser.Username, changePasswordParams.Password, dbUser.Role)
	if err != nil {
		logger.WebUILog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	filter := bson.M{"username": newPasswordDbUser.Username}
	_, err = dbadapter.WebuiDBClient.RestfulAPIPost(configmodels.UserAccountDataColl, filter, configmodels.ToBsonM(newPasswordDbUser))
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

var isFirstAccountIssued = func() (bool, error) {
	numOfUserAccounts, err := dbadapter.WebuiDBClient.RestfulAPICount(configmodels.UserAccountDataColl, bson.M{})
	if err != nil {
		return false, err
	}
	return numOfUserAccounts > 0, nil
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
