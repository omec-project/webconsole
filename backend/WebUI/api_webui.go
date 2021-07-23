// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package WebUI

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/free5gc/MongoDBLibrary"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/webconsole/backend/logger"
	"github.com/free5gc/webconsole/backend/webui_context"
	gServ "github.com/omec-project/webconsole/proto/server"
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

func mapToByte(data map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}

func sliceToByte(data []map[string]interface{}) (ret []byte) {
	ret, _ = json.Marshal(data)
	return
}

func toBsonM(data interface{}) (ret bson.M) {
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &ret)
	return
}

func toBsonA(data interface{}) (ret bson.A) {
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &ret)
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

	var subsData SubsData

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
			"free5gc",
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

	subsData = SubsData{
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

// Get all subscribers list
func GetSubscribers(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get All Subscribers List")

	var subsList []SubsListIE = make([]SubsListIE, 0)
	amDataList := MongoDBLibrary.RestfulAPIGetMany(amDataColl, bson.M{})
	for _, amData := range amDataList {
		ueId := amData["ueId"]
		servingPlmnId := amData["servingPlmnId"]
		tmp := SubsListIE{
			PlmnID: servingPlmnId.(string),
			UeId:   ueId.(string),
		}
		subsList = append(subsList, tmp)
	}

	c.JSON(http.StatusOK, subsList)
}

// Get subscriber by IMSI(ueId))
func GetSubscriberByID(c *gin.Context) {
	setCorsHeader(c)

	logger.WebUILog.Infoln("Get One Subscriber Data")

	var subsData SubsData

	ueId := c.Param("ueId")

	filterUeIdOnly := bson.M{"ueId": ueId}

	authSubsDataInterface := MongoDBLibrary.RestfulAPIGetOne(authSubsDataColl, filterUeIdOnly)
	amDataDataInterface := MongoDBLibrary.RestfulAPIGetOne(amDataColl, filterUeIdOnly)
	smDataDataInterface := MongoDBLibrary.RestfulAPIGetMany(smDataColl, filterUeIdOnly)
	smfSelDataInterface := MongoDBLibrary.RestfulAPIGetOne(smfSelDataColl, filterUeIdOnly)
	amPolicyDataInterface := MongoDBLibrary.RestfulAPIGetOne(amPolicyDataColl, filterUeIdOnly)
	smPolicyDataInterface := MongoDBLibrary.RestfulAPIGetOne(smPolicyDataColl, filterUeIdOnly)

	var authSubsData models.AuthenticationSubscription
	json.Unmarshal(mapToByte(authSubsDataInterface), &authSubsData)
	var amDataData models.AccessAndMobilitySubscriptionData
	json.Unmarshal(mapToByte(amDataDataInterface), &amDataData)
	var smDataData []models.SessionManagementSubscriptionData
	json.Unmarshal(sliceToByte(smDataDataInterface), &smDataData)
	var smfSelData models.SmfSelectionSubscriptionData
	json.Unmarshal(mapToByte(smfSelDataInterface), &smfSelData)
	var amPolicyData models.AmPolicyData
	json.Unmarshal(mapToByte(amPolicyDataInterface), &amPolicyData)
	var smPolicyData models.SmPolicyData
	json.Unmarshal(mapToByte(smPolicyDataInterface), &smPolicyData)

	subsData = SubsData{
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

// Post subscriber by IMSI(ueId)
func PostSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Post One Subscriber Data")

	var subsOverrideData SubsOverrideData
	if err := c.ShouldBindJSON(&subsOverrideData); err != nil {
		logger.WebUILog.Panic(err.Error())
	}

	ueId := c.Param("ueId")

	filterUeIdOnly := bson.M{"ueId": ueId}

	// start to compose a default UE info
	var servingPlmnId string
	servingPlmnId = "20893"
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
			OpcValue:            "8e27b6af0e692e750f32667a3b14605d", // Required
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862", // Required
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
		},
		SubscribedUeAmbr: &models.AmbrRm{
			Downlink: "2 Gbps",
			Uplink:   "1 Gbps",
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
						DefaultSscMode: models.SscMode__1,
						AllowedSscModes: []models.SscMode{
							"SSC_MODE_2",
							"SSC_MODE_3",
						},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "100 Mbps",
						Uplink:   "200 Mbps",
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
				},
				"internet2": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode: models.SscMode__1,
						AllowedSscModes: []models.SscMode{
							"SSC_MODE_2",
							"SSC_MODE_3",
						},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "100 Mbps",
						Uplink:   "200 Mbps",
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
						DefaultSscMode: models.SscMode__1,
						AllowedSscModes: []models.SscMode{
							"SSC_MODE_2",
							"SSC_MODE_3",
						},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "100 Mbps",
						Uplink:   "200 Mbps",
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
				},
				"internet2": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode: models.SscMode__1,
						AllowedSscModes: []models.SscMode{
							"SSC_MODE_2",
							"SSC_MODE_3",
						},
					},
					SessionAmbr: &models.Ambr{
						Downlink: "100 Mbps",
						Uplink:   "200 Mbps",
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
					{
						Dnn: "internet2",
					},
				},
			},
			"01112233": {
				DnnInfos: []models.DnnInfo{
					{
						Dnn: "internet",
					},
					{
						Dnn: "internet2",
					},
				},
			},
		},
	}

	amPolicyData := models.AmPolicyData{
		SubscCats: []string{
			"free5gc",
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
					"internet2": {
						Dnn: "internet2",
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
					"internet2": {
						Dnn: "internet2",
					},
				},
			},
		},
	}
	// end to compose a default UE info

	// override values
	if subsOverrideData.PlmnID != "" {
		servingPlmnId = subsOverrideData.PlmnID
	}
	if subsOverrideData.OPc != "" {
		authSubsData.Opc.OpcValue = subsOverrideData.OPc
	}
	if subsOverrideData.Key != "" {
		authSubsData.PermanentKey.PermanentKeyValue = subsOverrideData.Key
	}
	if subsOverrideData.SequenceNumber != "" {
		authSubsData.SequenceNumber = subsOverrideData.SequenceNumber
	}
	//	if subsOverrideData.DNN != nil {
	// TODO
	//	}

	authSubsBsonM := toBsonM(authSubsData)
	authSubsBsonM["ueId"] = ueId

	amDataBsonM := toBsonM(amDataData)
	amDataBsonM["ueId"] = ueId
	amDataBsonM["servingPlmnId"] = servingPlmnId

	smDatasBsonA := make([]interface{}, 0, len(smDataData))
	for _, smSubsData := range smDataData {
		smDataBsonM := toBsonM(smSubsData)
		smDataBsonM["ueId"] = ueId
		smDataBsonM["servingPlmnId"] = servingPlmnId
		smDatasBsonA = append(smDatasBsonA, smDataBsonM)
	}

	smfSelSubsBsonM := toBsonM(smfSelData)
	smfSelSubsBsonM["ueId"] = ueId
	smfSelSubsBsonM["servingPlmnId"] = servingPlmnId
	amPolicyDataBsonM := toBsonM(amPolicyData)
	amPolicyDataBsonM["ueId"] = ueId
	smPolicyDataBsonM := toBsonM(smPolicyData)
	smPolicyDataBsonM["ueId"] = ueId

	// there is no flowRule table in DB, this part of code is not used, uncomment it when need.
	/*	flowRulesBsonA := make([]interface{}, 0, len(subsData.FlowRules))
		for _, flowRule := range subsData.FlowRules {
			flowRuleBsonM := toBsonM(flowRule)
			flowRuleBsonM["ueId"] = ueId
			flowRuleBsonM["servingPlmnId"] = servingPlmnId
			flowRulesBsonA = append(flowRulesBsonA, flowRuleBsonM)
		}
	*/
	MongoDBLibrary.RestfulAPIPost(authSubsDataColl, filterUeIdOnly, authSubsBsonM)
	MongoDBLibrary.RestfulAPIPost(amDataColl, filterUeIdOnly, amDataBsonM)
	MongoDBLibrary.RestfulAPIPostMany(smDataColl, filterUeIdOnly, smDatasBsonA)
	MongoDBLibrary.RestfulAPIPost(smfSelDataColl, filterUeIdOnly, smfSelSubsBsonM)
	MongoDBLibrary.RestfulAPIPost(amPolicyDataColl, filterUeIdOnly, amPolicyDataBsonM)
	MongoDBLibrary.RestfulAPIPost(smPolicyDataColl, filterUeIdOnly, smPolicyDataBsonM)
	//	MongoDBLibrary.RestfulAPIPostMany(flowRuleDataColl, filterUeIdOnly, flowRulesBsonA)

	c.JSON(http.StatusCreated, gin.H{})
	gServ.HandleSubscriberAdd(ueId)
}

// Put subscriber by IMSI(ueId) and PlmnID(servingPlmnId)
func PutSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Put One Subscriber Data")

	var subsData SubsData
	if err := c.ShouldBindJSON(&subsData); err != nil {
		logger.WebUILog.Panic(err.Error())
	}

	ueId := c.Param("ueId")
	servingPlmnId := c.Param("servingPlmnId")

	filterUeIdOnly := bson.M{"ueId": ueId}
	filter := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId}

	authSubsBsonM := toBsonM(subsData.AuthenticationSubscription)
	authSubsBsonM["ueId"] = ueId
	amDataBsonM := toBsonM(subsData.AccessAndMobilitySubscriptionData)
	amDataBsonM["ueId"] = ueId
	amDataBsonM["servingPlmnId"] = servingPlmnId

	// Replace all data with new one
	MongoDBLibrary.RestfulAPIDeleteMany(smDataColl, filter)
	for _, data := range subsData.SessionManagementSubscriptionData {
		smDataBsonM := toBsonM(data)
		smDataBsonM["ueId"] = ueId
		smDataBsonM["servingPlmnId"] = servingPlmnId
		filterSmData := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId, "snssai": data.SingleNssai}
		MongoDBLibrary.RestfulAPIPutOne(smDataColl, filterSmData, smDataBsonM)
	}

	smfSelSubsBsonM := toBsonM(subsData.SmfSelectionSubscriptionData)
	smfSelSubsBsonM["ueId"] = ueId
	smfSelSubsBsonM["servingPlmnId"] = servingPlmnId
	amPolicyDataBsonM := toBsonM(subsData.AmPolicyData)
	amPolicyDataBsonM["ueId"] = ueId
	smPolicyDataBsonM := toBsonM(subsData.SmPolicyData)
	smPolicyDataBsonM["ueId"] = ueId

	MongoDBLibrary.RestfulAPIPutOne(authSubsDataColl, filterUeIdOnly, authSubsBsonM)
	MongoDBLibrary.RestfulAPIPutOne(amDataColl, filter, amDataBsonM)
	MongoDBLibrary.RestfulAPIPutOne(smfSelDataColl, filter, smfSelSubsBsonM)
	MongoDBLibrary.RestfulAPIPutOne(amPolicyDataColl, filterUeIdOnly, amPolicyDataBsonM)
	MongoDBLibrary.RestfulAPIPutOne(smPolicyDataColl, filterUeIdOnly, smPolicyDataBsonM)

	c.JSON(http.StatusNoContent, gin.H{})
	gServ.HandleSubscriberAdd(ueId)
}

// Patch subscriber by IMSI(ueId) and PlmnID(servingPlmnId)
func PatchSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Patch One Subscriber Data")

	var subsData SubsData
	if err := c.ShouldBindJSON(&subsData); err != nil {
		logger.WebUILog.Panic(err.Error())
	}

	ueId := c.Param("ueId")
	servingPlmnId := c.Param("servingPlmnId")

	filterUeIdOnly := bson.M{"ueId": ueId}
	filter := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId}

	authSubsBsonM := toBsonM(subsData.AuthenticationSubscription)
	authSubsBsonM["ueId"] = ueId
	amDataBsonM := toBsonM(subsData.AccessAndMobilitySubscriptionData)
	amDataBsonM["ueId"] = ueId
	amDataBsonM["servingPlmnId"] = servingPlmnId

	// Replace all data with new one
	MongoDBLibrary.RestfulAPIDeleteMany(smDataColl, filter)
	for _, data := range subsData.SessionManagementSubscriptionData {
		smDataBsonM := toBsonM(data)
		smDataBsonM["ueId"] = ueId
		smDataBsonM["servingPlmnId"] = servingPlmnId
		filterSmData := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId, "snssai": data.SingleNssai}
		MongoDBLibrary.RestfulAPIMergePatch(smDataColl, filterSmData, smDataBsonM)
	}

	smfSelSubsBsonM := toBsonM(subsData.SmfSelectionSubscriptionData)
	smfSelSubsBsonM["ueId"] = ueId
	smfSelSubsBsonM["servingPlmnId"] = servingPlmnId
	amPolicyDataBsonM := toBsonM(subsData.AmPolicyData)
	amPolicyDataBsonM["ueId"] = ueId
	smPolicyDataBsonM := toBsonM(subsData.SmPolicyData)
	smPolicyDataBsonM["ueId"] = ueId

	MongoDBLibrary.RestfulAPIMergePatch(authSubsDataColl, filterUeIdOnly, authSubsBsonM)
	MongoDBLibrary.RestfulAPIMergePatch(amDataColl, filter, amDataBsonM)
	MongoDBLibrary.RestfulAPIMergePatch(smfSelDataColl, filter, smfSelSubsBsonM)
	MongoDBLibrary.RestfulAPIMergePatch(amPolicyDataColl, filterUeIdOnly, amPolicyDataBsonM)
	MongoDBLibrary.RestfulAPIMergePatch(smPolicyDataColl, filterUeIdOnly, smPolicyDataBsonM)

	c.JSON(http.StatusNoContent, gin.H{})
}

// Delete subscriber by IMSI(ueId)
func DeleteSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Delete One Subscriber Data")

	ueId := c.Param("ueId")

	filterUeIdOnly := bson.M{"ueId": ueId}

	MongoDBLibrary.RestfulAPIDeleteOne(authSubsDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteOne(amDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteMany(smDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteMany(flowRuleDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteOne(smfSelDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteOne(amPolicyDataColl, filterUeIdOnly)
	MongoDBLibrary.RestfulAPIDeleteOne(smPolicyDataColl, filterUeIdOnly)

	c.JSON(http.StatusNoContent, gin.H{})
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

func compareNssai(sNssai *models.Snssai,
	sliceId *models.Snssai) int {
	if sNssai.Sst != sliceId.Sst {
		return 1
	}
	return strings.Compare(sNssai.Sd, sliceId.Sd)
}

func convertToString(val uint32) string {
	var mbVal, gbVal, kbVal uint32
	kbVal = val / 1024
	mbVal = val / 1048576
	gbVal = val / 1073741824
	var retStr string
	if gbVal != 0 {
		retStr = strconv.FormatUint(uint64(gbVal), 10) + " Gbps"
	} else if mbVal != 0 {
		retStr = strconv.FormatUint(uint64(mbVal), 10) + " Mbps"
	} else if kbVal != 0 {
		retStr = strconv.FormatUint(uint64(kbVal), 10) + " Kbps"
	} else {
		retStr = strconv.FormatUint(uint64(val), 10) + " bps"
	}

	return retStr
}

// SubscriptionUpdateHandle : Handle subscription update
func SubscriptionUpdateHandle(subsUpdateChan chan *gServ.SubsUpdMsg) {
	for subsData := range subsUpdateChan {
		logger.WebUILog.Infoln("SubscriptionUpdateHandle")
		var smDataData []models.SessionManagementSubscriptionData
		var smDatasBsonA []interface{}
		filterEmpty := bson.M{}
		var ueID string
		for _, ueID = range subsData.UeIds {
			filter := bson.M{"ueId": ueID}
			smDataDataInterface := MongoDBLibrary.RestfulAPIGetMany(smDataColl, filter)
			var found bool = false
			json.Unmarshal(sliceToByte(smDataDataInterface), &smDataData)
			if len(smDataData) != 0 {
				smDatasBsonA = make([]interface{}, 0, len(smDataData))
				for _, data := range smDataData {
					if compareNssai(data.SingleNssai, &subsData.Nssai) == 0 {
						logger.WebUILog.Infoln("entry exists for Imsi :  with SST:  and SD: ",
							ueID, subsData.Nssai.Sst, subsData.Nssai.Sd)
						found = true
						break
					}
				}

				if !found {
					logger.WebUILog.Infoln("entry doesnt exist for Imsi : %v with SST: %v and SD: %v",
						ueID, subsData.Nssai.Sst, subsData.Nssai.Sd)
					data := smDataData[0]
					data.SingleNssai.Sst = subsData.Nssai.Sst
					data.SingleNssai.Sd = subsData.Nssai.Sd
					data.SingleNssai.Sd = subsData.Nssai.Sd
					for idx, dnnCfg := range data.DnnConfigurations {
						var sessAmbr models.Ambr
						sessAmbr.Uplink = convertToString(uint32(subsData.Qos.Uplink))
						sessAmbr.Downlink = convertToString(uint32(subsData.Qos.Downlink))
						dnnCfg.SessionAmbr = &sessAmbr
						data.DnnConfigurations[idx] = dnnCfg
						logger.WebUILog.Infoln("uplink mbr ", data.DnnConfigurations[idx].SessionAmbr.Uplink)
						logger.WebUILog.Infoln("downlink mbr ", data.DnnConfigurations[idx].SessionAmbr.Downlink)
					}
					smDataBsonM := toBsonM(data)
					smDataBsonM["ueId"] = ueID
					smDataBsonM["servingPlmnId"] = subsData.ServingPlmnId
					logger.WebUILog.Infoln("servingplmnid ", subsData.ServingPlmnId)
					smDatasBsonA = append(smDatasBsonA, smDataBsonM)
				}
			} else {
				logger.WebUILog.Infoln("No imsi entry in db for imsi ", ueID)
			}
		}

		if len(smDatasBsonA) != 0 {
			MongoDBLibrary.RestfulAPIPostMany(smDataColl, filterEmpty, smDatasBsonA)
		}
	}
}
