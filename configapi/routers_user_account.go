// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/backend/auth"
)

func AddUserAccountService(engine *gin.Engine, jwtSecret []byte) {
	group := engine.Group("/config/v1")
	addRoutes(group, getUserAccountRoutes(jwtSecret))
}

func getUserAccountRoutes(jwtSecret []byte) Routes {
	return Routes{
		{
			"GetUserAccounts",
			http.MethodGet,
			"/account",
			auth.AdminOnly(jwtSecret, GetUserAccounts),
		},
		{
			"GetUserAccount",
			http.MethodGet,
			"/account/:username",
			auth.AdminOrMe(jwtSecret, GetUserAccount),
		},
		{
			"CreateUserAccount",
			http.MethodPost,
			"/account",
			auth.AdminOrFirstUser(jwtSecret, CreateUserAccount),
		},
		{
			"DeleteUserAccount",
			http.MethodDelete,
			"/account/:username",
			auth.AdminOnly(jwtSecret, DeleteUserAccount),
		},
		{
			"ChangeUserAccountPasssword",
			http.MethodPost,
			"/account/:username/change_password",
			auth.AdminOrMe(jwtSecret, ChangeUserAccountPasssword),
		},
	}
}
