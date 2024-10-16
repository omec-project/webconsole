// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type jwtGocertClaims struct {
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}

func GenerateJWTSecret() ([]byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return bytes, fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	return bytes, nil
}

// AdminOrUserAuthMiddleware intercepts requests that need authorization to check if the user's token exists and is
// permitted to use the endpoint
func AdminOrUserAuthMiddleware(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := getClaimsFromAuthorizationHeader(c.Request.Header.Get("Authorization"), jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("auth failed: %s", err.Error())})
			c.Abort()
			return
		}
		if claims.Role != configmodels.AdminRole && claims.Role != configmodels.UserRole {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "forbidden: admin or user access required"})
			c.Abort()
		}
		c.Next()
	}
}

// AdminOnly check if the authorization token is valid for this endpoint.
// Only tokens with AdminRole will be allowed.
func AdminOnly(jwtSecret []byte, handler func(c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		claims, err := getClaimsFromAuthorizationHeader(c.Request.Header.Get("Authorization"), jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("auth failed: %s", err.Error())})
			c.Abort()
			return
		}
		if claims.Role != configmodels.AdminRole {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin access required"})
			c.Abort()
			return
		}
		handler(c)
	}
}

// AdminOrMe check if the authorization token is valid for this endpoint.
// Admin role is allowed. UserRole is allowed with the condition of performing the action
// over their own account
func AdminOrMe(jwtSecret []byte, handler func(c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		claims, err := getClaimsFromAuthorizationHeader(c.Request.Header.Get("Authorization"), jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("auth failed: %s", err.Error())})
			c.Abort()
			return
		}
		if claims.Role == configmodels.AdminRole || (claims.Role == configmodels.UserRole && claims.Username == c.Param("username")) {
			handler(c)
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin or me access required"})
		c.Abort()
	}
}

// AdminOrFirstUser check if the authorization token is valid for this endpoint.
// check if the user has admin role or if the user is the first user before allowing access to the handler.
func AdminOrFirstUser(jwtSecret []byte, handler func(c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		numOfUserAccounts, err := dbadapter.WebuiDBClient.RestfulAPICount(configmodels.UserAccountDataColl, bson.M{})
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authorize"})
			c.Abort()
			return
		}
		if numOfUserAccounts > 0 {
			claims, err := getClaimsFromAuthorizationHeader(c.Request.Header.Get("Authorization"), jwtSecret)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("auth failed: %s", err.Error())})
				c.Abort()
				return
			}
			if claims.Role != configmodels.AdminRole {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin access required"})
				c.Abort()
				return
			}
		}
		handler(c)
	}
}

func getClaimsFromAuthorizationHeader(header string, JwtSecret []byte) (*jwtGocertClaims, error) {
	if header == "" {
		return nil, fmt.Errorf("authorization header not found")
	}
	bearerToken := strings.Split(header, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		return nil, fmt.Errorf("authorization header couldn't be processed. The expected format is 'Bearer token'")
	}
	claims, err := getClaimsFromJWT(bearerToken[1], JwtSecret)
	if err != nil {
		return nil, fmt.Errorf("token is not valid")
	}
	return claims, nil
}

func getClaimsFromJWT(bearerToken string, JwtSecret []byte) (*jwtGocertClaims, error) {
	claims := jwtGocertClaims{}
	token, err := jwt.ParseWithClaims(bearerToken, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return &claims, nil
}
