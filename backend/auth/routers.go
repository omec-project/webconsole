// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Route is the information for every URI.
type LoginRoute struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFuncWithSecret func(jwtSecret []byte) gin.HandlerFunc
}

type Route struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFunc gin.HandlerFunc
}

type LoginRoutes []LoginRoute

type Routes []Route

func AddService(engine *gin.Engine, jwtSecret []byte) {
	addLoginService(engine, jwtSecret)
	addUserManagementService(engine)
}

func addLoginService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/")
	for _, route := range rootRoutes {
		handler := route.HandlerFuncWithSecret(jwtSecret)
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, handler)
		case "POST":
			group.POST(route.Pattern, handler)
		case "PUT":
			group.PUT(route.Pattern, handler)
		case "DELETE":
			group.DELETE(route.Pattern, handler)
		}
	}
}

func addUserManagementService(engine *gin.Engine) {
	group := engine.Group("/config/v1")
	addService(group, routes)
}

func addService(group *gin.RouterGroup, routes Routes) {
	for _, route := range routes {
		//handler := route.HandlerFunc
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, route.HandlerFunc)
		case "POST":
			group.POST(route.Pattern, route.HandlerFunc)
		case "PUT":
			group.PUT(route.Pattern, route.HandlerFunc)
		case "DELETE":
			group.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
}

var rootRoutes = LoginRoutes{
	{
		"Login",
		http.MethodPost,
		"/login",
		Login,
	},
}

var routes = Routes{
	{
		"GetUserAccounts",
		http.MethodGet,
		"/account",
		GetUserAccounts,
	},
	{
		"GetUserAccount",
		http.MethodGet,
		"/account/:username",
		GetUserAccount,
	},
	{
		"PostUserAccount",
		http.MethodPost,
		"/account",
		PostUserAccount,
	},
	{
		"DeleteUserAccount",
		http.MethodDelete,
		"/account/:username",
		DeleteUserAccount,
	},
	{
		"ChangeUserAccountPasssword",
		http.MethodPost,
		"/account/:username/change_password",
		ChangeUserAccountPasssword,
	},
}
