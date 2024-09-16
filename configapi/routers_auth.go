// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/auth"
)

// Route is the information for every URI.
type AuthRoute struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFuncWithSecret func(jwtSecret []byte) gin.HandlerFunc
}

type AuthRoutes []AuthRoute

func AddAuthService(engine *gin.Engine, jwtSecret []byte) {
	addLoginService(engine, jwtSecret)
	AddService(engine, "/config/v1", userManagementRoutes)
}

func addLoginService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/")
	for _, route := range loginRoutes {
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

var loginRoutes = AuthRoutes{
	{
		"Login",
		http.MethodPost,
		"/login",
		auth.Login,
	},
}

var userManagementRoutes = Routes{
	{
		Name:        "GetUserAccounts",
		Method:      http.MethodGet,
		Pattern:     "/account",
		HandlerFunc: auth.GetUserAccounts,
	},
	{
		Name:        "GetUserAccount",
		Method:      http.MethodGet,
		Pattern:     "/account/:username",
		HandlerFunc: auth.GetUserAccount,
	},
	{
		Name:        "PostUserAccount",
		Method:      http.MethodPost,
		Pattern:     "/account",
		HandlerFunc: auth.PostUserAccount,
	},
	{
		Name:        "DeleteUserAccount",
		Method:      http.MethodDelete,
		Pattern:     "/account/:username",
		HandlerFunc: auth.DeleteUserAccount,
	},
	{
		Name:        "ChangeUserAccountPasssword",
		Method:      http.MethodPost,
		Pattern:     "/account/:username/change_password",
		HandlerFunc: auth.ChangeUserAccountPasssword,
	},
}
