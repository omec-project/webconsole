// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package authentication

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
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

func mapToByte(data map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}
func toBsonM(data interface{}) (ret bson.M) {
	tmp, err := json.Marshal(data)
	if err != nil {
		logger.DbLog.Errorln("Could not marshall data")
		return nil
	}
	err = json.Unmarshal(tmp, &ret)
	if err != nil {
		logger.DbLog.Errorln("Could not unmarshall data")
		return nil
	}
	return ret
}

func GetUserAccounts(c *gin.Context) {
	users, err := FetchUserAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func FetchUserAccounts() ([]*configmodels.User, error) {
	rawUsers, errGetMany := dbadapter.UserAccountDBClient.RestfulAPIGetMany(userAccountDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorln(errGetMany.Error())
		return nil, fmt.Errorf(errorRetrieveUserAccounts)
	}
	var users []*configmodels.User
	users = make([]*configmodels.User, 0)
	for _, rawUser := range rawUsers {
		var userData configmodels.User
		err := json.Unmarshal(mapToByte(rawUser), &userData)
		if err != nil {
			logger.AuthLog.Errorf(errorRetrieveUserAccount)
			continue
		}
		userData.Password = ""
		users = append(users, &userData)
	}
	return users, nil
}

func IsFirstAccountIssued() (bool, error) {
	users, err := FetchUserAccounts()
	if err != nil {
		return false, err
	}
	return len(users) > 0, nil
}

func GetUserAccount(c *gin.Context) {
	logger.WebUILog.Infoln("get user account")
	var err error
	username := c.Param("username")
	filter := bson.M{"username": username}
	rawUser, err := dbadapter.UserAccountDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if len(rawUser) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	var userAccount configmodels.User
	err = json.Unmarshal(mapToByte(rawUser), &userAccount)
	if err != nil {
		logger.AuthLog.Errorln(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	userAccount.Password = ""
	c.JSON(http.StatusOK, userAccount)
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
	var shouldGeneratePassword = user.Password == ""
	if shouldGeneratePassword {
		generatedPassword, err := generatePassword()
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
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
	filter := bson.M{"username": user.Username}
	rawUser, err := dbadapter.UserAccountDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if len(rawUser) != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user account already exists"})
		return
	}

	rawUsers, err := dbadapter.UserAccountDBClient.RestfulAPIGetMany(userAccountDataColl, bson.M{})
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccounts})
		return
	}
	user.Permissions = USER_ACCOUNT
	if len(rawUsers) == 0 {
		logger.DbLog.Errorln(len(rawUsers))
		user.Permissions = ADMIN_ACCOUNT //if this is the first user it will be admin
	}
	password := user.Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorCreateUserAccount})
		return
	}
	user.Password = string(hashedPassword)
	userBsonA := toBsonM(user)

	_, err = dbadapter.UserAccountDBClient.RestfulAPIPost(userAccountDataColl, filter, userBsonA)
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
	filter := bson.M{"username": username}
	rawUser, err := dbadapter.UserAccountDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if len(rawUser) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	var userAccount configmodels.User
	err = json.Unmarshal(mapToByte(rawUser), &userAccount)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if userAccount.Permissions == 1 {
		logger.AuthLog.Errorln(errorDeleteAdminAccount)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorDeleteAdminAccount})
		return
	}
	errDelOne := dbadapter.UserAccountDBClient.RestfulAPIDeleteOne(userAccountDataColl, filter)
	if errDelOne != nil {
		logger.DbLog.Errorln(errDelOne)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorDeleteUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func ChangeUserAccountPasssword(c *gin.Context) {
	logger.WebUILog.Infoln("change user password")
	username := c.Param("username")
	var userAccount configmodels.User
	err := c.ShouldBindJSON(&userAccount)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
		return
	}
	if userAccount.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
		return
	}
	if !validatePassword(userAccount.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidPassword})
		return
	}
	filter := bson.M{"username": username}
	rawUser, err := dbadapter.UserAccountDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	if len(rawUser) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": errorUsernameNotFound})
		return
	}
	var postUser configmodels.User
	err = json.Unmarshal(mapToByte(rawUser), &postUser)
	if err != nil {
		logger.AuthLog.Errorln(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
		return
	}
	password := userAccount.Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.AuthLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	postUser.Password = string(hashedPassword)
	userBsonA := toBsonM(postUser)
	_, err = dbadapter.UserAccountDBClient.RestfulAPIPost(userAccountDataColl, filter, userBsonA)
	if err != nil {
		logger.DbLog.Errorln(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorUpdateUserAccount})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func Login(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userRequest configmodels.User
		err := c.ShouldBindJSON(&userRequest)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": errorInvalidDataProvided})
			return
		}
		if userRequest.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingUsername})
			return
		}
		if userRequest.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMissingPassword})
			return
		}

		filter := bson.M{"username": userRequest.Username}
		rawUser, err := dbadapter.UserAccountDBClient.RestfulAPIGetOne(userAccountDataColl, filter)
		if err != nil {
			logger.DbLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if len(rawUser) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		var userAccount configmodels.User
		err = json.Unmarshal(mapToByte(rawUser), &userAccount)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorRetrieveUserAccount})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(userAccount.Password), []byte(userRequest.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorIncorrectCredentials})
			return
		}
		jwt, err := generateJWT(userAccount.Username, userAccount.Permissions, jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": errorLogin})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": jwt})
	}
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
	if !hasCapital {
		return false
	}
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	if !hasLower {
		return false
	}
	hasNumberOrSymbol := regexp.MustCompile(`[0-9!@#$%^&*()_+\-=\[\]{};':"|,.<>?~]`).MatchString(password)

	return hasNumberOrSymbol
}
