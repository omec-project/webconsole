// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type Routes []Route

func AddAuthenticationService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/")
	addRoutes(group, getAuthenticationRoutes(jwtSecret))
}

func addRoutes(group *gin.RouterGroup, routes Routes) {
	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			group.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			group.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			group.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			group.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
}

func getAuthenticationRoutes(jwtSecret []byte) Routes {
	return Routes{
		{
			"Login",
			http.MethodPost,
			"/login",
			Login(jwtSecret),
		},
		{
			"Status",
			http.MethodGet,
			"/status",
			GetStatus(),
		},
	}
}
