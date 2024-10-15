// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configmodels

import (
	"golang.org/x/crypto/bcrypt"
)

type DBUserAccount struct {
	Username       string `json:"username"`
	HashedPassword string `json:"password,omitempty"`
	Role           int    `json:"role"`
}

type CreateUserAccountParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangePasswordParams struct {
	Password string `json:"password"`
}

type GetUserAccountResponse struct {
	Username string `json:"username"`
	Role     int    `json:"role"`
}

func CreateNewDBUserAccount(username string, password string, role int) (*DBUserAccount, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	dbUser := &DBUserAccount{
		Username:       username,
		HashedPassword: string(hashedPassword),
		Role:           role,
	}
	return dbUser, nil
}
