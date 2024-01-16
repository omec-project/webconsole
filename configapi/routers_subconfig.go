// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package configapi

import (
	"github.com/omec-project/util/mongoapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddServiceSub(engine *gin.Engine, m mongoapi.MongoClient) *gin.RouterGroup {
	group := engine.Group("/api")
	var routesL = GetRoutesList(m)

	for _, route := range routesL {
		switch route.Method {
		case http.MethodGet:
			group.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			group.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			group.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			group.DELETE(route.Pattern, route.HandlerFunc)
		case http.MethodPatch:
			group.PATCH(route.Pattern, route.HandlerFunc)
		}
	}

	return group
}

func GetRoutesList(m mongoapi.MongoClient) Routes {
	return Routes{
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
			GetSubscribers(m),
		},

		{
			"GetSubscriberByID",
			http.MethodGet,
			"/subscriber/:ueId",
			GetSubscriberByID(m),
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
			"/subscriber/:ueId/:servingPlmnId",
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
			"/subscriber/:ueId/:servingPlmnId",
			PatchSubscriberByID,
		},

		{
			"Registered UE Context",
			http.MethodGet,
			"/registered-ue-context",
			GetRegisteredUEContext(m),
		},

		{
			"Individual Registered UE Context",
			http.MethodGet,
			"/registered-ue-context/:supi",
			GetRegisteredUEContext(m),
		},

		{
			"UE PDU Session Info",
			http.MethodGet,
			"/ue-pdu-session-info/:smContextRef",
			GetUEPDUSessionInfo(m),
		},
	}
}
