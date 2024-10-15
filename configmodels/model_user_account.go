// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package configmodels

import (
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Role     int    `json:"role"`
}

type DBUser struct {
	Username       string `json:"username"`
	HashedPassword string `json:"password,omitempty"`
	Role           int    `json:"role"`
}

type CreateUserParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     int    `json:"role"`
}

type UserResponse struct {
	Username string `json:"username"`
	Role     int    `json:"role"`
}

func TransformDBUserToUserResponse(dbUser DBUser) UserResponse {
	return UserResponse{
		Username: dbUser.Username,
		Role:     dbUser.Role,
	}
}

func TransformDBUsersToUserResponses(dbUsers []*DBUser) []*UserResponse {
	userResponses := make([]*UserResponse, len(dbUsers))

	for i, dbUser := range dbUsers {
		userResponses[i] = &UserResponse{
			Username: dbUser.Username,
			Role:     dbUser.Role,
		}
	}

	return userResponses
}

func TransformCreateUserParamsToDBUser(params CreateUserParams) (*DBUser, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	dbUser := &DBUser{
		Username:       params.Username,
		HashedPassword: string(hashedPassword),
		Role:           params.Role,
	}
	return dbUser, nil
}
