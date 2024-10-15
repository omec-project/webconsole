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
)

const (
	errorCreateUserAccount    = "failed to create user account"
	errorDeleteAdminAccount   = "deleting an admin user account is not allowed"
	errorDeleteUserAccount    = "failed to delete user account"
	errorIncorrectCredentials = "incorrect username or password. Try again"
	errorInvalidDataProvided  = "invalid data provided"
	errorInvalidPassword      = "Password must have 8 or more characters, must include at least one capital letter, one lowercase letter, and either a number or a symbol."
	errorMissingPassword      = "password is required"
	errorMissingUsername      = "username is required"
	errorRetrieveUserAccount  = "failed to retrieve user account from DB"
	errorRetrieveUserAccounts = "failed to retrieve user accounts from DB"
	errorUpdateUserAccount    = "failed to update user account"
	errorUsernameNotFound     = "username not found"
	UserAccountDataColl       = "webconsoleData.snapshots.userAccountData"
)

func GetUserAccounts(c *gin.Context) {
	dbUsersAccounts, err := fetchDBUsers()
	userResponses := make([]*configmodels.GetUserAccountResponse, len(dbUsersAccounts))

	for i, dbUserAccount := range dbUsersAccounts {
		userResponses[i] = &configmodels.GetUserAccountResponse{
			Username: dbUserAccount.Username,
			Role:     dbUserAccount.Role,
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, userResponses)
}

func fetchDBUsers() ([]*configmodels.DBUserAccount, error) {
	rawUsers, err := dbadapter.WebuiDBClient.RestfulAPIGetMany(UserAccountDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, errors.New(errorRetrieveUserAccounts)
	}
	var users []*configmodels.DBUserAccount
	users = make([]*configmodels.DBUserAccount, 0)
	for _, rawUser := range rawUsers {
		var user configmodels.DBUserAccount
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
	rawUserAccount, err := dbadapter.WebuiDBClient.RestfulAPIGetOne(UserAccountDataColl, filter)
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
		logger.AuthLog.Errorln(err.Error())
		return nil, err
	}
	return &userAccount, nil
}

func CreateUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("create user account")
	var createUserParams configmodels.CreateUserAccountParams
	err := c.ShouldBindJSON(&createUserParams)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
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
	newUserRole := UserRole
	isFirstAccountIssued, err := IsFirstAccountIssued()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	if !isFirstAccountIssued {
		newUserRole = AdminRole
	}
	dbUser, err := configmodels.CreateNewDBUserAccount(createUserParams.Username, createUserParams.Password, newUserRole)
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
	dbUserAccount, err := fetchDBUserAccount(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUserAccount == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	if dbUserAccount.Role == AdminRole {
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
	var changePasswordParams configmodels.ChangePasswordParams
	err := c.ShouldBindJSON(&changePasswordParams)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
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
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	filter := bson.M{"username": newPasswordDbUser.Username}
	_, err = dbadapter.WebuiDBClient.RestfulAPIPost(UserAccountDataColl, filter, configmodels.ToBsonM(newPasswordDbUser))
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

var IsFirstAccountIssued = func() (bool, error) {
	numOfUserAccounts, err := dbadapter.WebuiDBClient.RestfulAPICount(UserAccountDataColl, bson.M{})
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
