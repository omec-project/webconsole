// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddApiService(engine *gin.Engine, middlewares ...gin.HandlerFunc) *gin.RouterGroup {
	group := engine.Group("/api")
	if len(middlewares) > 0 {
		group.Use(middlewares...)
	}
	addRoutes(group, apiRoutes)
	return group
}

var apiRoutes = Routes{
	{
		"GetExample",
		http.MethodGet,
		"/sample",
		GetSampleJSON,
	},

	{
		"GetSubscribers",
		http.MethodGet,
		"/subscriber",
		GetSubscribers,
	},

	{
		"GetSubscriberByID",
		http.MethodGet,
		"/subscriber/:ueId",
		GetSubscriberByID,
	},

	{
		"PostSubscriberByID",
		http.MethodPost,
		"/subscriber/:ueId",
		PostSubscriberByID,
	},

	{
		"PutSubscriberByID",
		http.MethodPut,
		"/subscriber/:ueId",
		PutSubscriberByID,
	},

	{
		"DeleteSubscriberByID",
		http.MethodDelete,
		"/subscriber/:ueId",
		DeleteSubscriberByID,
	},

	{
		"PatchSubscriberByID",
		http.MethodPatch,
		"/subscriber/:ueId",
		PatchSubscriberByID,
	},

	{
		"Registered UE Context",
		http.MethodGet,
		"/registered-ue-context",
		GetRegisteredUEContext,
	},

	{
		"Individual Registered UE Context",
		http.MethodGet,
		"/registered-ue-context/:supi",
		GetRegisteredUEContext,
	},

	{
		"UE PDU Session Info",
		http.MethodGet,
		"/ue-pdu-session-info/:smContextRef",
		GetUEPDUSessionInfo,
	},
	// K4 api endpoint (CRUD)
	{
		"Get k4 keys",
		http.MethodGet,
		"/k4opt",
		HandleGetsK4,
	},
	{
		"Get a only k4 keys filtering using the sno",
		http.MethodGet,
		"/k4opt/:idsno",
		HandleGetK4,
	},
	{
		"Post k4 key to create a k4 key",
		http.MethodPost,
		"/k4opt",
		HandlePostK4,
	},
	{
		"Update k4 keys",
		http.MethodPut,
		"/k4opt/:idsno",
		HandlePutK4,
	},
	{
		"Delete k4 keys",
		http.MethodDelete,
		"/k4opt/:idsno/:keylabel",
		HandleDeleteK4,
	},
}
