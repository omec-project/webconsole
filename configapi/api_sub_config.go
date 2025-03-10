// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/webui_context"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	authSubsDataColl = "subscriptionData.authenticationData.authenticationSubscription"
	amDataColl       = "subscriptionData.provisionedData.amData"
	smDataColl       = "subscriptionData.provisionedData.smData"
	smfSelDataColl   = "subscriptionData.provisionedData.smfSelectionSubscriptionData"
	amPolicyDataColl = "policyData.ues.amData"
	smPolicyDataColl = "policyData.ues.smData"
	flowRuleDataColl = "policyData.ues.flowRule"
)

var httpsClient *http.Client

func init() {
	httpsClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func sliceToByte(data []map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}

func setCorsHeader(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
}

func sendResponseToClient(c *gin.Context, response *http.Response) {
	var jsonData interface{}
	json.NewDecoder(response.Body).Decode(&jsonData)
	c.JSON(response.StatusCode, jsonData)
}

func GetSampleJSON(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get a JSON Example")

	var subsData configmodels.SubsData

	authSubsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA", // "5G_AKA", "EAP_AKA_PRIME"
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "c9e8763286b5b9ffbdf56e1297d0887b", // Required
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            "981d464c7c52eb6e5036234984ad0bcf", // Required
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "5122250214c33e723a5dd523fc145fc0", // Required
		},
		SequenceNumber: "16f3b3f70fc2",
	}

	amDataData := models.AccessAndMobilitySubscriptionData{
		Gpsis: []string{
			"msisdn-0900000000",
		},
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{
				{
					Sd:  "010203",
					Sst: 1,
				},
				{
					Sd:  "112233",
					Sst: 1,
				},
			},
			SingleNssais: []models.Snssai{
				{
					Sd:  "010203",
					Sst: 1,
				},
				{
					Sd:  "112233",
					Sst: 1,
				},
			},
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: "1000 Kbps",
			Uplink:   "1000 Kbps",
		},
	}

	smDataData := []models.SessionManagementSubscriptionData{
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd:  "010203",
			},
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode:  models.SscMode__1,
						AllowedSscModes: []models.SscMode{models.SscMode__1},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "1000 Kbps",
						Uplink:   "1000 Kbps",
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
				},
			},
		},
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd:  "112233",
			},
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode:  models.SscMode__1,
						AllowedSscModes: []models.SscMode{models.SscMode__1},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "1000 Kbps",
						Uplink:   "1000 Kbps",
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
				},
			},
		},
	}

	smfSelData := models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			"01010203": {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: "internet",
					},
				},
			},
			"01112233": {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: "internet",
					},
				},
			},
		},
	}

	amPolicyData := models.AmPolicyData{
		SubscCats: []string{
			"aether",
		},
	}

	smPolicyData := models.SmPolicyData{
		SmPolicySnssaiData: map[string]models.SmPolicySnssaiData{
			"01010203": {
				Snssai: &models.Snssai{
					Sd:  "010203",
					Sst: 1,
				},
				SmPolicyDnnData: map[string]models.SmPolicyDnnData{
					"internet": {
						Dnn: "internet",
					},
				},
			},
			"01112233": {
				Snssai: &models.Snssai{
					Sd:  "112233",
					Sst: 1,
				},
				SmPolicyDnnData: map[string]models.SmPolicyDnnData{
					"internet": {
						Dnn: "internet",
					},
				},
			},
		},
	}

	servingPlmnId := "20893"
	ueId := "imsi-2089300007487"

	subsData = configmodels.SubsData{
		PlmnID:                            servingPlmnId,
		UeId:                              ueId,
		AuthenticationSubscription:        authSubsData,
		AccessAndMobilitySubscriptionData: amDataData,
		SessionManagementSubscriptionData: smDataData,
		SmfSelectionSubscriptionData:      smfSelData,
		AmPolicyData:                      amPolicyData,
		SmPolicyData:                      smPolicyData,
	}
	c.JSON(http.StatusOK, subsData)
}

// GetSubscribers godoc
//
// @Description  Return the list of subscribers
// @Tags         Subscribers
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  configmodels.SubsListIE  "List of subscribers. Null if there are no subscribers"
// @Failure      401  {object}  nil                      "Authorization failed"
// @Failure      403  {object}  nil                      "Forbidden"
// @Failure      500  {object}  nil                      "Error retrieving subscribers"
// @Router      /api/subscriber/  [get]
func GetSubscribers(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get All Subscribers List")

	subsList := make([]configmodels.SubsListIE, 0)
	amDataList, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(amDataColl, bson.M{})
	if errGetMany != nil {
		logger.DbLog.Errorw("failed to retrieve subscribers list", "error", errGetMany)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscribers list"})
		return
	}
	for _, amData := range amDataList {

		tmp := configmodels.SubsListIE{
			UeId: amData["ueId"].(string),
		}

		if servingPlmnId, plmnIdExists := amData["servingPlmnId"]; plmnIdExists {
			tmp.PlmnID = servingPlmnId.(string)
		}

		subsList = append(subsList, tmp)
	}

	c.JSON(http.StatusOK, subsList)
}

// GetSubscriberByID godoc
//
// @Description  Get subscriber by IMSI (UE ID)
// @Tags         Subscribers
// @Param        imsi    path    string    true    "IMSI (UE ID)"    example(imsi-208930100007487)
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  nil  "Subscriber"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      404  {object}  nil  "Subscriber not found"
// @Failure      500  {object}  nil  "Error retrieving subscriber"
// @Router      /api/subscriber/{imsi}  [get]
func GetSubscriberByID(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get One Subscriber Data")

	ueId := c.Param("ueId")
	filterUeIdOnly := bson.M{"ueId": ueId}

	var subsData configmodels.SubsData

	authSubsDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(authSubsDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch authentication subscription data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	amDataDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(amDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch am data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	smDataDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetMany(smDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch sm data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	smfSelDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(smfSelDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch smf selection data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	amPolicyDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(amPolicyDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch am policy data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	smPolicyDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(smPolicyDataColl, filterUeIdOnly)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch sm policy data from DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested user record from DB"})
		return
	}
	// If all fetched data is empty, return 404 error
	if authSubsDataInterface == nil &&
		amDataDataInterface == nil &&
		smDataDataInterface == nil &&
		smfSelDataInterface == nil &&
		amPolicyDataInterface == nil &&
		smPolicyDataInterface == nil {
		logger.WebUILog.Errorf("subscriber with ID %s not found", ueId)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("subscriber with ID %s not found", ueId)})
		return
	}

	var authSubsData models.AuthenticationSubscription
	if authSubsDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(authSubsDataInterface), &authSubsData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling authentication subscription data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	var amDataData models.AccessAndMobilitySubscriptionData
	if amDataDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(amDataDataInterface), &amDataData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling access and mobility subscription data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	var smDataData []models.SessionManagementSubscriptionData
	if smDataDataInterface != nil {
		err := json.Unmarshal(sliceToByte(smDataDataInterface), &smDataData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling session management subscription data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	var smfSelData models.SmfSelectionSubscriptionData
	if smfSelDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(smfSelDataInterface), &smfSelData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling smf selection subscription data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	var amPolicyData models.AmPolicyData
	if amPolicyDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(amPolicyDataInterface), &amPolicyData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling am policy data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	var smPolicyData models.SmPolicyData
	if smPolicyDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(smPolicyDataInterface), &smPolicyData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling sm policy data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
			return
		}
	}

	subsData = configmodels.SubsData{
		UeId:                              ueId,
		AuthenticationSubscription:        authSubsData,
		AccessAndMobilitySubscriptionData: amDataData,
		SessionManagementSubscriptionData: smDataData,
		SmfSelectionSubscriptionData:      smfSelData,
		AmPolicyData:                      amPolicyData,
		SmPolicyData:                      smPolicyData,
	}

	c.JSON(http.StatusOK, subsData)
}

// PostSubscriberByID godoc
//
// @Description  Create subscriber by IMSI (UE ID)
// @Tags         Subscribers
// @Param        imsi       path    string                           true    "IMSI (UE ID)"
// @Param        content    body    configmodels.SubsOverrideData    true    " "
// @Security     BearerAuth
// @Success      201  {object}  nil  "Subscriber created"
// @Failure      400  {object}  nil  "Invalid subscriber content"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      409  {object}  nil  "Subscriber already exists"
// @Failure      500  {object}  nil  "Error creating subscriber"
// @Router      /api/subscriber/{imsi}  [post]
func PostSubscriberByID(c *gin.Context) {
	setCorsHeader(c)

	var subsOverrideData configmodels.SubsOverrideData
	if err := c.ShouldBindJSON(&subsOverrideData); err != nil {
		logger.WebUILog.Errorln("Post One Subscriber Data - ShouldBindJSON failed ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ueId := c.Param("ueId")

	logger.WebUILog.Infoln("Received Post Subscriber Data from Roc/Simapp: ", ueId)

	// Check if the IMSI already exists in the database
	filter := bson.M{"ueId": ueId}
	subscriber, err := dbadapter.CommonDBClient.RestfulAPIGetOne(amDataColl, filter)
	if err != nil {
		logger.DbLog.Errorf("Failed querying subscriber existence for IMSI: %s; Error: %v", ueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check subscriber: %s existence", ueId)})
		return
	} else if subscriber != nil {
		logger.WebUILog.Errorf("subscriber %s already exists", ueId)
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("subscriber %s already exists", ueId)})
		return
	}
	authSubsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA", // "5G_AKA", "EAP_AKA_PRIME"
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "", // Required
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			// OpcValue:            "8e27b6af0e692e750f32667a3b14605d", // Required
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			// PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862", // Required
		},
		// SequenceNumber: "16f3b3f70fc2",
	}

	// override values
	/*if subsOverrideData.PlmnID != "" {
		servingPlmnId = subsOverrideData.PlmnID
	}*/
	if subsOverrideData.OPc != "" {
		authSubsData.Opc.OpcValue = subsOverrideData.OPc
	}
	if subsOverrideData.Key != "" {
		authSubsData.PermanentKey.PermanentKeyValue = subsOverrideData.Key
	}
	if subsOverrideData.SequenceNumber != "" {
		authSubsData.SequenceNumber = subsOverrideData.SequenceNumber
	}
	c.JSON(http.StatusCreated, gin.H{})

	msg := configmodels.ConfigMessage{
		MsgType:     configmodels.Sub_data,
		MsgMethod:   configmodels.Post_op,
		AuthSubData: &authSubsData,
		Imsi:        ueId,
	}
	configChannel <- &msg
	logger.WebUILog.Infoln("Successfully Added Subscriber Data to ConfigChannel: ", ueId)
}

// Put subscriber by IMSI(ueId) and PlmnID(servingPlmnId)
func PutSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Put One Subscriber Data")

	var subsData configmodels.SubsData
	if err := c.ShouldBindJSON(&subsData); err != nil {
		logger.WebUILog.Panic(err.Error())
	}

	ueId := c.Param("ueId")
	c.JSON(http.StatusNoContent, gin.H{})

	msg := configmodels.ConfigMessage{
		MsgType:     configmodels.Sub_data,
		MsgMethod:   configmodels.Post_op,
		AuthSubData: &subsData.AuthenticationSubscription,
		Imsi:        ueId,
	}
	configChannel <- &msg
	logger.WebUILog.Infoln("Put Subscriber Data complete")
}

// Patch subscriber by IMSI(ueId) and PlmnID(servingPlmnId)
func PatchSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Patch One Subscriber Data")
}

// DeleteSubscriberByID godoc
//
// @Description  Delete an existing subscriber
// @Tags         Subscribers
// @Param        imsi    path    string    true    "IMSI (UE ID)"
// @Security     BearerAuth
// @Success      204  {object}  nil  "Subscriber deleted successfully"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      500  {object}  nil  "Error deleting subscriber"
// @Router       /api/subscriber/{imsi}  [delete]
func DeleteSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Delete One Subscriber Data")

	ueId := c.Param("ueId")

	imsi := strings.TrimPrefix(ueId, "imsi-")
	err := updateSubscriberInDeviceGroups(imsi)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error deleting subscriber"})
		return
	}

	msg := configmodels.ConfigMessage{
		MsgType:   configmodels.Sub_data,
		MsgMethod: configmodels.Delete_op,
		Imsi:      ueId,
	}
	configChannel <- &msg
	c.JSON(http.StatusNoContent, gin.H{})
	logger.WebUILog.Infoln("Delete Subscriber Data complete")
}

func updateSubscriberInDeviceGroups(imsi string) error {
	filterByImsi := bson.M{
		"imsis": imsi,
	}
	rawDeviceGroups, err := dbadapter.CommonDBClient.RestfulAPIGetMany(devGroupDataColl, filterByImsi)
	if err != nil {
		logger.DbLog.Errorf("failed to fetch device groups: %v", err)
		return err
	}
	var deviceGroupUpdateMessages []configmodels.ConfigMessage
	for _, rawDeviceGroup := range rawDeviceGroups {
		var deviceGroup configmodels.DeviceGroups
		if err = json.Unmarshal(configmodels.MapToByte(rawDeviceGroup), &deviceGroup); err != nil {
			logger.DbLog.Errorf("error unmarshaling device group: %v", err)
			return err
		}
		filteredImsis := []string{}
		for _, currImsi := range deviceGroup.Imsis {
			if currImsi != imsi {
				filteredImsis = append(filteredImsis, currImsi)
			}
		}
		deviceGroup.Imsis = filteredImsis
		deviceGroupUpdateMessage := configmodels.ConfigMessage{
			MsgType:      configmodels.Device_group,
			MsgMethod:    configmodels.Post_op,
			DevGroupName: deviceGroup.DeviceGroupName,
			DevGroup:     &deviceGroup,
		}
		deviceGroupUpdateMessages = append(deviceGroupUpdateMessages, deviceGroupUpdateMessage)
	}
	for _, msg := range deviceGroupUpdateMessages {
		configChannel <- &msg
		logger.WebUILog.Infof("device group [%v] update sent to config channel", msg.DevGroupName)
	}
	return nil
}

func GetRegisteredUEContext(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get Registered UE Context")

	webuiSelf := webui_context.WEBUI_Self()
	webuiSelf.UpdateNfProfiles()

	supi, supiExists := c.Params.Get("supi")

	// TODO: support fetching data from multiple AMFs
	if amfUris := webuiSelf.GetOamUris(models.NfType_AMF); amfUris != nil {
		var requestUri string

		if supiExists {
			requestUri = fmt.Sprintf("%s/namf-oam/v1/registered-ue-context/%s", amfUris[0], supi)
		} else {
			requestUri = fmt.Sprintf("%s/namf-oam/v1/registered-ue-context", amfUris[0])
		}

		resp, err := httpsClient.Get(requestUri)
		if err != nil {
			logger.WebUILog.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		sendResponseToClient(c, resp)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"cause": "No AMF Found",
		})
	}
}

func GetUEPDUSessionInfo(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get UE PDU Session Info")

	webuiSelf := webui_context.WEBUI_Self()
	webuiSelf.UpdateNfProfiles()

	smContextRef, smContextRefExists := c.Params.Get("smContextRef")
	if !smContextRefExists {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}

	// TODO: support fetching data from multiple SMF
	if smfUris := webuiSelf.GetOamUris(models.NfType_SMF); smfUris != nil {
		requestUri := fmt.Sprintf("%s/nsmf-oam/v1/ue-pdu-session-info/%s", smfUris[0], smContextRef)
		resp, err := httpsClient.Get(requestUri)
		if err != nil {
			logger.WebUILog.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}

		sendResponseToClient(c, resp)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"cause": "No SMF Found",
		})
	}
}
