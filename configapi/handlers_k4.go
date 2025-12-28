package configapi

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	ssmapi "github.com/omec-project/webconsole/configapi/ssm_api"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

// HandleGetsK4 retrieves all K4 keys from the database.
//
// This handler processes GET requests to /k4opt endpoint and returns a list of all K4 keys
// stored in the MongoDB database. Each K4 key contains both the key value and its
// associated sequence number (SNO).
//
// Parameters:
//   - c (*gin.Context): The Gin context containing the HTTP request and response.
//
// Returns:
//   - 200 OK: Successfully retrieved the list of K4 keys.
//   - 500 Internal Server Error: If there was an error retrieving the data from the database.
//
// Example Response:
//
//	[
//	  {
//	    "k4": "abc123def456",
//	    "k4_sno": 1
//	  },
//	  {
//	    "k4": "xyz789def456",
//	    "k4_sno": 2
//	  }
//	]
func HandleGetsK4(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get All K4 keys List")

	k4List := make([]configmodels.K4, 0)
	k4DataList, errGetMany := dbadapter.AuthDBClient.RestfulAPIGetMany(K4KeysColl, bson.M{})
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve k4 keys list with error: %+v", errGetMany)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve k4 keys list"})
		return
	}

	var k4Data configmodels.K4
	for _, k4DataInterface := range k4DataList {
		err := json.Unmarshal(configmodels.MapToByte(k4DataInterface), &k4Data)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling k4 key data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve k4 key"})
			return
		}

		k4List = append(k4List, k4Data)
	}

	c.JSON(http.StatusOK, k4List)
}

// HandleGetK4 retrieves a specific K4 key by its sequence number (SNO).
//
// This handler processes GET requests to /k4opt/:idsno endpoint where :idsno is the
// sequence number of the K4 key to retrieve. It returns a single K4 key object if found.
//
// Parameters:
//   - c (*gin.Context): The Gin context containing the HTTP request and response.
//   - idsno (path parameter): The sequence number of the K4 key to retrieve.
//
// Returns:
//   - 200 OK: Successfully retrieved the K4 key.
//   - 500 Internal Server Error: If there was an error retrieving the data from the database.
//
// Example Response:
//
//	{
//	  "k4": "abc123def456",
//	  "k4_sno": 1
//	}
func HandleGetK4(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get One K4 key Data")

	snoId := c.Param("idsno")
	snoIdint, _ := strconv.Atoi(snoId)

	filterSnoID := bson.M{"k4_sno": snoIdint}

	var k4Data configmodels.K4

	k4DataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(K4KeysColl, filterSnoID)

	if err != nil {
		logger.AppLog.Errorf("failed to fetch k4 key data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested k4 key record from DB"})
		return
	}

	if k4DataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(k4DataInterface), &k4Data)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling k4 key data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve k4 key"})
			return
		}
	}

	c.JSON(http.StatusOK, k4Data)
}

// HandlePostK4 creates a new K4 key in the database.
//
// This handler processes POST requests to /k4opt endpoint. It accepts a JSON body
// containing the K4 key data and stores it in the database. The K4 key must have
// a unique sequence number (SNO).
//
// Parameters:
//   - c (*gin.Context): The Gin context containing the HTTP request and response.
//
// Request Body:
//
//	{
//	  "k4": "abc123def456",    // The K4 key value
//	  "k4_sno": 1             // The sequence number for the key
//	}
//
// Returns:
//   - 201 Created: Successfully created the K4 key.
//   - 400 Bad Request: If the request body is invalid or cannot be parsed.
//   - 500 Internal Server Error: If there was an error storing the data in the database.
//
// Example Response:
// Returns the created K4 key object with HTTP status 201.
func HandlePostK4(c *gin.Context) {
	setCorsHeader(c)

	// TODO: if the k4 is plain, remove the k4 value for empty string ""

	logger.WebUILog.Infoln("Post One K4 key Data")

	var k4Data configmodels.K4
	var err error

	rawData, err := c.GetRawData()
	if err != nil {
		logger.WebUILog.Errorf("failed to get raw data: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get raw data"})
		return
	}

	logger.WebUILog.Infof("Raw data received: %s", string(rawData))

	err = json.Unmarshal(rawData, &k4Data)
	if err != nil {
		logger.WebUILog.Errorf("failed to unmarshall the json: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshall the json"})
		return
	}

	// validate data posted
	if k4Data.K4_SNO == 0 {
		logger.WebUILog.Errorln("K4_SNO is missing or zero in the request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "K4_SNO must be provided and greater than zero"})
		return
	}

	if _, err := hex.DecodeString(k4Data.K4); err != nil {
		logger.WebUILog.Errorf("K4 key is not a valid hex string: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "K4 key must be a valid hex string"})
		return
	}
	// end validate data posted

	// Normalize K4 to lowercase
	k4Data.K4 = strings.ToLower(k4Data.K4)

	logger.WebUILog.Infof("Parsed K4 data: %+v", k4Data)

	logger.WebUILog.Infof("K4 data to be inserted: %+v", k4Data)

	// SSM
	// Store the K4 in the SSM if this option is allow
	if factory.WebUIConfig.Configuration.SSM.AllowSsm {
		if err := ssmapi.Ssmhsm_api.StoreKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to store k4 key in SSM: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store k4 key in SSM"})
			return
		}
	}

	// Vault
	// Store the K4 in Vault if this option is enabled
	if factory.WebUIConfig.Configuration.Vault != nil && factory.WebUIConfig.Configuration.Vault.AllowVault {
		if err := ssmapi.Vault_api.StoreKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to store k4 key in Vault: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store k4 key in Vault"})
			return
		}
	}

	k4Data.TimeCreated = time.Now()
	k4Data.TimeUpdated = k4Data.TimeCreated

	// MongoDB
	// Save the K4 data in MongoDB
	if err := K4HelperPost(int(k4Data.K4_SNO), &k4Data); err != nil {
		logger.AppLog.Errorf("failed to post k4 key in DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to post k4 key"})
		return
	}

	logger.WebUILog.Infoln("K4 key posted successfully")
	c.JSON(http.StatusCreated, k4Data)
}

// HandlePutK4 updates an existing K4 key in the database.
//
// This handler processes PUT requests to /k4opt/:idsno endpoint where :idsno is the
// sequence number of the K4 key to update. It accepts a JSON body containing the new
// K4 key data and updates the existing record in the database.
//
// Parameters:
//   - c (*gin.Context): The Gin context containing the HTTP request and response.
//   - idsno (path parameter): The sequence number of the K4 key to update.
//
// Request Body:
//
//	{
//	  "k4": "xyz789def456",    // The new K4 key value
//	  "k4_sno": 1             // Must match the idsno in the URL
//	}
//
// Returns:
//   - 200 OK: Successfully updated the K4 key.
//   - 400 Bad Request: If the request body is invalid or cannot be parsed.
//   - 500 Internal Server Error: If there was an error updating the data in the database.
//
// Example Response:
// Returns the updated K4 key object with HTTP status 200.
func HandlePutK4(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Put One K4 key Data")

	snoId := c.Param("idsno")
	snoIdint, _ := strconv.Atoi(snoId)
	var k4Data configmodels.K4

	if err := c.ShouldBindJSON(&k4Data); err != nil {
		logger.WebUILog.Errorf("Put One K4 key Data - ShouldBindJSON failed: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: failed to parse JSON."})
		return
	}

	// validate data update
	if k4Data.K4_SNO == 0 {
		logger.WebUILog.Errorln("K4_SNO is missing or zero in the request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "K4_SNO must be provided and greater than zero"})
		return
	}

	if _, err := hex.DecodeString(k4Data.K4); err != nil {
		logger.WebUILog.Errorf("K4 key is not a valid hex string: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "K4 key must be a valid hex string"})
		return
	}
	// end validate data update

	// Normalize K4 to lowercase
	k4Data.K4 = strings.ToLower(k4Data.K4)

	// SSM
	// Update the K4 in the SSM if this option is allow
	if factory.WebUIConfig.Configuration.SSM.AllowSsm {
		if err := ssmapi.Ssmhsm_api.UpdateKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to update k4 key in SSM: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update k4 key in SSM"})
			return
		}
	}

	// Vault
	// Update the K4 in Vault if this option is enabled
	if factory.WebUIConfig.Configuration.Vault != nil && factory.WebUIConfig.Configuration.Vault.AllowVault {
		if err := ssmapi.Vault_api.UpdateKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to update k4 key in Vault: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update k4 key in Vault"})
			return
		}
	}

	k4Data.TimeCreated = time.Now()
	k4Data.TimeUpdated = k4Data.TimeCreated

	if err := K4HelperPut(snoIdint, &k4Data); err != nil {
		logger.AppLog.Errorf("failed to update k4 key in DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update k4 key"})
		return
	}

	c.JSON(http.StatusOK, k4Data)
}

// HandleDeleteK4 removes a K4 key from the database.
//
// This handler processes DELETE requests to /k4opt/:idsno endpoint where :idsno is the
// sequence number of the K4 key to delete. It removes both the K4 key and its associated
// data from the database.
//
// Parameters:
//   - c (*gin.Context): The Gin context containing the HTTP request and response.
//   - idsno (path parameter): The sequence number of the K4 key to delete.
//
// Returns:
//   - 200 OK: Successfully deleted the K4 key.
//   - 500 Internal Server Error: If there was an error deleting the data from the database.
//
// Example Response:
//
//	{
//	  "message": "k4 key deleted successfully"
//	}
func HandleDeleteK4(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Delete One K4 key Data")

	snoId := c.Param("idsno")
	keylabel := c.Param("keylabel")
	snoIdint, _ := strconv.Atoi(snoId)

	k4Data := configmodels.K4{
		K4_Label: keylabel,
		K4_SNO:   byte(snoIdint),
	}

	// SSM
	// Delete the K4 in the SSM if this option is allow
	if factory.WebUIConfig.Configuration.SSM.AllowSsm {
		if err := ssmapi.Ssmhsm_api.DeleteKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to delete k4 key in SSM: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete k4 key in SSM"})
			return
		}
	}

	// Vault
	// Delete the K4 in Vault if this option is enabled
	if factory.WebUIConfig.Configuration.Vault != nil && factory.WebUIConfig.Configuration.Vault.AllowVault {
		if err := ssmapi.Vault_api.DeleteKey(&k4Data); err != nil {
			logger.AppLog.Errorf("failed to delete k4 key in Vault: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete k4 key in Vault"})
			return
		}
	}

	if err := K4HelperDelete(snoIdint, keylabel); err != nil {
		logger.AppLog.Errorf("failed to delete k4 key in DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete k4 key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "k4 key deleted successfully"})
}
