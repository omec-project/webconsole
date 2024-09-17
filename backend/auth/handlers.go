// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	mrand "math/rand"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

const (
	errorCreateUserAccount    = "failed to create user"
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
	errorUpdateUserAccount    = "failed to update user"
	errorUsernameNotFound     = "username not found"

	userAccountDataColl = "webconsoleData.snapshots.userAccountData"
)

func GetUserAccounts(c *gin.Context) {
	users, err := fetchUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func fetchUsers() ([]*configmodels.User, error) {
	rawUsers, err := dbadapter.WebuiDBClient.RestfulAPIGetMany(userAccountDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, errors.New(errorRetrieveUserAccounts)
	}
	var users []*configmodels.User
	users = make([]*configmodels.User, 0)
	for _, rawUser := range rawUsers {
		var user configmodels.User
		err := json.Unmarshal(configmodels.MapToByte(rawUser), &user)
		if err != nil {
			logger.DbLog.Errorf(errorRetrieveUserAccount)
			continue
		}
		user.Password = ""
		users = append(users, &user)
	}
	return users, nil
}

func GetUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("get user account")
	username := c.Param("username")
	user, err := fetchUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

func fetchUser(username string) (*configmodels.User, error) {
	filter := bson.M{"username": username}
	rawUser, err := dbadapter.WebuiDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		return nil, err
	}
	if len(rawUser) == 0 {
		return nil, nil
	}
	var user configmodels.User
	err = json.Unmarshal(configmodels.MapToByte(rawUser), &user)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		return nil, err
	}
	return &user, nil
}

func PostUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("create user account")
	var user configmodels.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if user.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
		return
	}
	shouldGeneratePassword := user.Password == ""
	if shouldGeneratePassword {
		generatedPassword, passwordErr := generatePassword()
		if passwordErr != nil {
			logger.AuthLog.Errorln(passwordErr.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
			return
		}
		user.Password = generatedPassword
	}
	if !validatePassword(user.Password) {
		logger.AuthLog.Errorln("invalid password provided")
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	dbUser, err := fetchUser(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user account already exists"})
		return
	}
	user.Permissions = USER_ACCOUNT
	isFirstAccountIssued, err := IsFirstAccountIssued()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	if !isFirstAccountIssued {
		user.Permissions = ADMIN_ACCOUNT
	}
	password := user.Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
		return
	}
	user.Password = string(hashedPassword)
	filter := bson.M{"username": user.Username}
	_, err = dbadapter.WebuiDBClient.RestfulAPIPost(userAccountDataColl, filter, configmodels.ToBsonM(user))
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
		return
	}
	if shouldGeneratePassword {
		c.JSON(http.StatusCreated, gin.H{"password": password})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

func DeleteUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("delete user account")
	username := c.Param("username")
	dbUser, err := fetchUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	if dbUser.Permissions == 1 {
		logger.AuthLog.Errorln(errorDeleteAdminAccount)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorDeleteAdminAccount})
		return
	}
	filter := bson.M{"username": username}
	err = dbadapter.WebuiDBClient.RestfulAPIDeleteOne(userAccountDataColl, filter)
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
	var user configmodels.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	dbUser, err := fetchUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if dbUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	dbUser.Password = string(hashedPassword)
	filter := bson.M{"username": dbUser.Username}
	_, err = dbadapter.WebuiDBClient.RestfulAPIPost(userAccountDataColl, filter, configmodels.ToBsonM(dbUser))
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func Login(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		var inputUser configmodels.User
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
		dbUser, err := fetchUser(inputUser.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if dbUser == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		if err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(inputUser.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		jwt, err := generateJWT(dbUser.Username, dbUser.Permissions, jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorLogin})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": jwt})
	}
}

func IsFirstAccountIssued() (bool, error) {
	users, err := fetchUsers()
	if err != nil {
		return false, err
	}
	return len(users) > 0, nil
}

// Generates a random 16 chars long password that contains uppercase and lowercase characters and numbers or symbols.
var generatePassword = func() (string, error) {
	const (
		uppercaseSet         = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercaseSet         = "abcdefghijklmnopqrstuvwxyz"
		numbersAndSymbolsSet = "0123456789*?@"
		allCharsSet          = uppercaseSet + lowercaseSet + numbersAndSymbolsSet
	)
	uppercase, err := getRandomChars(uppercaseSet, 2)
	if err != nil {
		return "", err
	}
	lowercase, err := getRandomChars(lowercaseSet, 2)
	if err != nil {
		return "", err
	}
	numbersOrSymbols, err := getRandomChars(numbersAndSymbolsSet, 2)
	if err != nil {
		return "", err
	}
	allChars, err := getRandomChars(allCharsSet, 10)
	if err != nil {
		return "", err
	}
	res := []rune(uppercase + lowercase + numbersOrSymbols + allChars)
	mrand.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})
	return string(res), nil
}

func getRandomChars(charset string, length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
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
