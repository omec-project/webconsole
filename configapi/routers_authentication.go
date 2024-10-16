// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddAuthenticationService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/")
	addRoutes(group, getAuthenticationRoutes(jwtSecret))
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
