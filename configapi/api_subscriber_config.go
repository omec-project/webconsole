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
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/webui_context"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type subscribersPageResponse struct {
	Items []configmodels.SubsListIE `json:"items"`
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
	Total int                       `json:"total"`
	Pages int                       `json:"pages"`
}

func parsePositiveIntQuery(c *gin.Context, name string, defaultValue int) (int, error) {
	valueStr := strings.TrimSpace(c.Query(name))
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return value, nil
}

func buildSubscribersFilter(c *gin.Context) bson.M {
	plmnID := strings.TrimSpace(c.Query("plmnID"))
	ueID := strings.TrimSpace(c.Query("ueId"))
	if ueID == "" {
		ueID = strings.TrimSpace(c.Query("imsi"))
	}
	q := strings.TrimSpace(c.Query("q"))

	andFilters := make([]bson.M, 0, 3)
	if plmnID != "" {
		andFilters = append(andFilters, bson.M{"servingPlmnId": plmnID})
	}
	if ueID != "" {
		andFilters = append(andFilters, bson.M{"ueId": ueID})
	}
	if q != "" {
		andFilters = append(andFilters, bson.M{"ueId": bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}})
	}

	switch len(andFilters) {
	case 0:
		return bson.M{}
	case 1:
		return andFilters[0]
	default:
		return bson.M{"$and": andFilters}
	}
}

func shouldReturnSubscribersMeta(c *gin.Context) bool {
	if strings.EqualFold(strings.TrimSpace(c.Query("withMeta")), "true") {
		return true
	}
	// Any query implies the client expects a structured response.
	for _, key := range []string{"page", "limit", "plmnID", "ueId", "imsi", "q"} {
		if strings.TrimSpace(c.Query(key)) != "" {
			return true
		}
	}
	return false
}

var httpsClient *http.Client

func init() {
	httpsClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func SliceToByte(data []map[string]any) ([]byte, error) {
	ret, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	return ret, nil
}

func setCorsHeader(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
}

func sendResponseToClient(c *gin.Context, response *http.Response) {
	var jsonData any
	if err := json.NewDecoder(response.Body).Decode(&jsonData); err != nil {
		logger.AppLog.Errorf("failed to decode response: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode response"})
		return
	}
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
			EncryptionKey:       "",
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

	useMeta := shouldReturnSubscribersMeta(c)
	filter := buildSubscribersFilter(c)

	page := 1
	limit := 50
	if useMeta {
		var err error
		page, err = parsePositiveIntQuery(c, "page", 1)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		limit, err = parsePositiveIntQuery(c, "limit", 50)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if limit > 500 {
			limit = 500
		}
	}

	subsList := make([]configmodels.SubsListIE, 0)
	amDataList, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany(AmDataColl, filter)
	if errGetMany != nil {
		logger.AppLog.Errorf("failed to retrieve subscribers list with error: %+v", errGetMany)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscribers list"})
		return
	}
	logger.AppLog.Infof("GetSubscribers: len: %d", len(amDataList))
	if len(amDataList) == 0 {
		if useMeta {
			c.JSON(http.StatusOK, subscribersPageResponse{Items: subsList, Page: page, Limit: limit, Total: 0, Pages: 0})
			return
		}
		c.JSON(http.StatusOK, subsList)
		return
	}
	for _, amData := range amDataList {
		var subsData configmodels.SubsListIE

		err := json.Unmarshal(configmodels.MapToByte(amData), &subsData)
		if err != nil {
			logger.AppLog.Errorf("could not unmarshal subscriber %s", amData)
		}

		if servingPlmnId, plmnIdExists := amData["servingPlmnId"]; plmnIdExists {
			subsData.PlmnID = servingPlmnId.(string)
		}

		subsList = append(subsList, subsData)
	}

	sort.SliceStable(subsList, func(i, j int) bool {
		return subsList[i].UeId < subsList[j].UeId
	})

	if !useMeta {
		c.JSON(http.StatusOK, subsList)
		return
	}

	total := len(subsList)
	if total == 0 {
		c.JSON(http.StatusOK, subscribersPageResponse{Items: []configmodels.SubsListIE{}, Page: page, Limit: limit, Total: 0, Pages: 0})
		return
	}

	pages := int(math.Ceil(float64(total) / float64(limit)))
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}

	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := int(math.Min(float64(total), float64(start+limit)))
	if end < start {
		end = start
	}

	items := subsList[start:end]
	// Ensure JSON "items" is never null.
	if items == nil {
		items = []configmodels.SubsListIE{}
	}

	c.JSON(http.StatusOK, subscribersPageResponse{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
		Pages: pages,
	})
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

	authSubsDataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(AuthSubsDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch authentication subscription data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
		return
	}
	amDataDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(AmDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch am data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
		return
	}
	smDataDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetMany(SmDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch sm data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
		return
	}
	smfSelDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(SmfSelDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch smf selection data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
		return
	}
	amPolicyDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(AmPolicyDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch am policy data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
		return
	}
	smPolicyDataInterface, err := dbadapter.CommonDBClient.RestfulAPIGetOne(SmPolicyDataColl, filterUeIdOnly)
	if err != nil {
		logger.AppLog.Errorf("failed to fetch sm policy data from DB: %+v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch the requested subscriber record from DB"})
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
			logger.WebUILog.Errorf("error unmarshalling authentication subscription data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
	}

	var amDataData models.AccessAndMobilitySubscriptionData
	if amDataDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(amDataDataInterface), &amDataData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling access and mobility subscription data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
	}

	var smDataData []models.SessionManagementSubscriptionData
	if smDataDataInterface != nil {
		bytesData, err := SliceToByte(smDataDataInterface)
		if err != nil {
			logger.WebUILog.Errorf("failed to convert slice to byte: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
		err = json.Unmarshal(bytesData, &smDataData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling session management subscription data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
	}

	var smfSelData models.SmfSelectionSubscriptionData
	if smfSelDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(smfSelDataInterface), &smfSelData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling smf selection subscription data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
	}

	var amPolicyData models.AmPolicyData
	if amPolicyDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(amPolicyDataInterface), &amPolicyData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling am policy data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
			return
		}
	}

	var smPolicyData models.SmPolicyData
	if smPolicyDataInterface != nil {
		err := json.Unmarshal(configmodels.MapToByte(smPolicyDataInterface), &smPolicyData)
		if err != nil {
			logger.WebUILog.Errorf("error unmarshalling sm policy data: %+v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve subscriber"})
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
	requestID := uuid.New().String()
	var subsOverrideData configmodels.SubsOverrideData
	if err := c.ShouldBindJSON(&subsOverrideData); err != nil {
		logger.WebUILog.Errorf("Post One Subscriber Data - ShouldBindJSON failed: %+v request ID: %s", err, requestID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: failed to parse JSON.", "request_id": requestID})
		return
	}
	logger.WebUILog.Infof("%+v", subsOverrideData)

	ueId := c.Param("ueId")
	if ueId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing ueId in request URL", "request_id": requestID})
		return
	}

	logger.WebUILog.Infoln("Received Post Subscriber Data from Roc/Simapp:", ueId)
	logger.WebUILog.Debugf("Override Data: %+v", subsOverrideData)

	// Check if the IMSI already exists in the database
	filter := bson.M{"ueId": ueId}
	subscriber, err := dbadapter.CommonDBClient.RestfulAPIGetOne(AmDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed querying subscriber existence for IMSI: %s; Error: %+v", ueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check subscriber: %s existence", ueId), "request_id": requestID})
		return
	} else if subscriber != nil {
		logger.WebUILog.Errorf("subscriber %s already exists", ueId)
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("subscriber %s already exists", ueId), "request_id": requestID})
		return
	}
	if subsOverrideData.OPc == "" || subsOverrideData.Key == "" || subsOverrideData.SequenceNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required authentication data: OPc and Key must be provided", "request_id": requestID})
		return
	}
	var ceroValue int32
	if subsOverrideData.EncryptionAlgorithm == nil {
		subsOverrideData.EncryptionAlgorithm = &ceroValue
	}
	if *subsOverrideData.EncryptionAlgorithm < 0 || *subsOverrideData.EncryptionAlgorithm > 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Encription Algoritm is not valid: Encription Algoritm must be between 0 and 4", "request_id": requestID})
		return
	}

	authSubsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "",
			},
		},
		Opc: &models.Opc{
			OpcValue:            subsOverrideData.OPc,
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
		},
		PermanentKey: &models.PermanentKey{
			PermanentKeyValue:   subsOverrideData.Key,
			EncryptionAlgorithm: *subsOverrideData.EncryptionAlgorithm,
			EncryptionKey:       "",
		},
		SequenceNumber: subsOverrideData.SequenceNumber,
	}

	if subsOverrideData.EncryptionAlgorithm != nil {
		authSubsData.PermanentKey.EncryptionAlgorithm = *subsOverrideData.EncryptionAlgorithm
	}
	if subsOverrideData.K4Sno != nil {
		authSubsData.K4_SNO = *subsOverrideData.K4Sno
	}

	if err := assingK4Key(subsOverrideData.K4Sno, &authSubsData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("Failed to create subscriber %s", ueId),
			"request_id": requestID,
			"message":    "Please refer to the log with the provided Request ID for details, error assing the K4 Key",
		})
		return
	}

	logger.WebUILog.Infof("%+v", authSubsData)
	logger.WebUILog.Infof("Using OPc: %s, Key: %s, SeqNo: %s", subsOverrideData.OPc, subsOverrideData.Key, subsOverrideData.SequenceNumber)

	err = SubscriberAuthenticationDataCreate(ueId, &authSubsData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("Failed to create subscriber %s", ueId),
			"request_id": requestID,
			"message":    "Please refer to the log with the provided Request ID for details",
		})
		return
	}
	logger.WebUILog.Infof("Subscriber %s created successfully", ueId)
	c.JSON(http.StatusCreated, gin.H{})
}

// PutSubscriberByID godoc
//
// @Description  Update subscriber information by IMSI (UE ID)
// @Tags         Subscribers
// @Param        imsi       path    string                           true    "IMSI (UE ID)"
// @Param        content    body    configmodels.SubsData            true    "Updated subscriber details"
// @Security     BearerAuth
// @Success      204  {object}  nil  "Subscriber updated successfully"
// @Failure      400  {object}  nil  "Invalid subscriber content"
// @Failure      401  {object}  nil  "Authorization failed"
// @Failure      403  {object}  nil  "Forbidden"
// @Failure      404  {object}  nil  "Subscriber not found"
// @Failure      500  {object}  nil  "Error updating subscriber"
// @Router       /api/subscriber/{imsi}  [put]
func PutSubscriberByID(c *gin.Context) {
	setCorsHeader(c)
	logger.WebUILog.Infoln("Put One Subscriber Data")
	setCorsHeader(c)
	requestID := uuid.New().String()
	var subsOverrideData configmodels.SubsOverrideData
	if err := c.ShouldBindJSON(&subsOverrideData); err != nil {
		logger.WebUILog.Errorf("Put One Subscriber Data - ShouldBindJSON failed: %+v request ID: %s", err, requestID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: failed to parse JSON.", "request_id": requestID})
		return
	}

	ueId := c.Param("ueId")
	logger.WebUILog.Infoln("Received Put Subscriber Data from Roc/Simapp:", ueId)

	filter := bson.M{"ueId": ueId}
	subscriber, err := dbadapter.CommonDBClient.RestfulAPIGetOne(AmDataColl, filter)
	if err != nil {
		logger.AppLog.Errorf("failed querying subscriber existence for IMSI: %s; Error: %+v", ueId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check subscriber: %s existence", ueId), "request_id": requestID})
		return
	}
	if subscriber == nil {
		logger.WebUILog.Errorf("subscriber %s does not exist", ueId)
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("subscriber %s does not exist", ueId)})
		return
	}
	if subsOverrideData.OPc == "" || subsOverrideData.Key == "" || subsOverrideData.SequenceNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required authentication data: OPc, Key and Sequence number must be provided", "request_id": requestID})
		return
	}
	var ceroValue int32
	if subsOverrideData.EncryptionAlgorithm == nil {
		subsOverrideData.EncryptionAlgorithm = &ceroValue
	}
	if *subsOverrideData.EncryptionAlgorithm < 0 || *subsOverrideData.EncryptionAlgorithm > 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Encription Algoritm is not valid: Encription Algoritm must be between 0 and 4", "request_id": requestID})
		return
	}
	authSubsData := models.AuthenticationSubscription{
		AuthenticationManagementField: "8000",
		AuthenticationMethod:          "5G_AKA",
		Milenage: &models.Milenage{
			Op: &models.Op{
				EncryptionAlgorithm: 0,
				EncryptionKey:       0,
				OpValue:             "",
			},
		},
		Opc: &models.Opc{
			EncryptionAlgorithm: 0,
			EncryptionKey:       0,
			OpcValue:            subsOverrideData.OPc,
		},
		PermanentKey: &models.PermanentKey{
			EncryptionAlgorithm: *subsOverrideData.EncryptionAlgorithm,
			EncryptionKey:       "",
			PermanentKeyValue:   subsOverrideData.Key,
		},
		SequenceNumber: subsOverrideData.SequenceNumber,
	}

	if subsOverrideData.EncryptionAlgorithm != nil {
		authSubsData.PermanentKey.EncryptionAlgorithm = *subsOverrideData.EncryptionAlgorithm
	}
	if subsOverrideData.K4Sno != nil {
		authSubsData.K4_SNO = *subsOverrideData.K4Sno
	} else {
		authSubsData.K4_SNO = 0
	}

	if err := assingK4Key(subsOverrideData.K4Sno, &authSubsData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("Failed to create subscriber %s", ueId),
			"request_id": requestID,
			"message":    "Please refer to the log with the provided Request ID for details, error assing the K4 Key",
		})
		return
	}

	err = SubscriberAuthenticationDataUpdate(ueId, &authSubsData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("Failed to update subscriber %s", ueId),
			"request_id": requestID,
			"message":    "Please refer to the log with the provided Request ID for details",
		})
		return
	}
	logger.WebUILog.Infof("Subscriber %s updated successfully", ueId)
	c.JSON(http.StatusNoContent, gin.H{})
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
	requestID := uuid.New().String()

	ueId := c.Param("ueId")

	imsi := strings.TrimPrefix(ueId, "imsi-")
	statusCode, err := updateSubscriberInDeviceGroupsWhenDeleteSub(imsi)
	if err != nil {
		logger.WebUILog.Errorf("Failed to update subscriber: %+v request ID: %s", err, requestID)
		c.JSON(statusCode, gin.H{"error": "error deleting subscriber. Please check the log for details.", "request_id": requestID})
		return
	}
	if err = subscriberAuthenticationDataDelete(ueId); err != nil {
		logger.WebUILog.Errorf("Error deleting subscriber: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("Failed to delete subscriber %s", ueId),
			"request_id": requestID,
			"message":    "Please refer to the log with the provided Request ID for details",
		})
		return
	}
	logger.WebUILog.Infof("Subscriber %s deleted successfully", ueId)

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

func assingK4Key(k4Sno *byte, authSubsData *models.AuthenticationSubscription) error {
	if k4Sno != nil {
		snoIdint := int(*k4Sno)
		filter := bson.M{"k4_sno": snoIdint}
		if factory.WebUIConfig.Configuration.SSM.AllowSsm {
			filter = bson.M{
				"key_label": ssm_constants.AlgorithmLabelMap[int(authSubsData.PermanentKey.EncryptionAlgorithm)],
				"k4_sno":    snoIdint,
			}
		}

		var k4Data configmodels.K4

		k4DataInterface, err := dbadapter.AuthDBClient.RestfulAPIGetOne(K4KeysColl, filter)

		if err != nil {
			logger.AppLog.Errorf("failed to fetch k4 key data from DB: %+v", err)
			return err
		}

		if k4DataInterface != nil {
			err := json.Unmarshal(configmodels.MapToByte(k4DataInterface), &k4Data)
			if err != nil {
				logger.WebUILog.Errorf("error unmarshalling k4 key data: %+v", err)
				return err
			}
		}

		authSubsData.PermanentKey.EncryptionKey = k4Data.K4
	}
	return nil
}
