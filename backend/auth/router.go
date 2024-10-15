// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Route represents a route for the service.
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type Routes []Route

// AddService adds all routes to the Gin engine.
func AddService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/")
	addRoutes(group, rootRoutesWithSecret(jwtSecret))

	group = engine.Group("/config/v1")
	addRoutes(group, rootRoutesWithAuthorization(jwtSecret))
}

// addRoutes adds routes to a Gin RouterGroup.
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

// rootRoutesWithSecret adds the JWT secret to the handler functions.
func rootRoutesWithSecret(jwtSecret []byte) Routes {
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
			GetStatus(jwtSecret),
		},
	}
}

func rootRoutesWithAuthorization(jwtSecret []byte) Routes {
	return Routes{
		{
			"GetUserAccounts",
			http.MethodGet,
			"/account",
			adminOnly(jwtSecret, GetUserAccounts),
		},
		{
			"GetUserAccount",
			http.MethodGet,
			"/account/:username",
			adminOrMe(jwtSecret, GetUserAccount),
		},
		{
			"CreateUserAccount",
			http.MethodPost,
			"/account",
			adminOrFirstUser(jwtSecret, CreateUserAccount),
		},
		{
			"DeleteUserAccount",
			http.MethodDelete,
			"/account/:username",
			adminOnly(jwtSecret, DeleteUserAccount),
		},
		{
			"ChangeUserAccountPasssword",
			http.MethodPost,
			"/account/:username/change_password",
			adminOrMe(jwtSecret, ChangeUserAccountPasssword),
		},
	}
}
