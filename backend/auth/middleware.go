// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

package auth

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/omec-project/webconsole/backend/logger"
)

const (
	USER_ACCOUNT  = 0
	ADMIN_ACCOUNT = 1
)

type jwtGocertClaims struct {
	Username    string `json:"username"`
	Permissions int    `json:"permissions"`
	jwt.StandardClaims
}

func GenerateJWTSecret() ([]byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return bytes, fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	return bytes, nil
}

var generateJWT = func(username string, permissions int, jwtSecret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtGocertClaims{
		Username:    username,
		Permissions: permissions,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
		},
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// authMiddleware intercepts requests that need authorization to check if the user's token exists and is
// permitted to use the endpoint
func AuthMiddleware(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/config/v1") && !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}
		if c.Request.Method == "POST" && strings.HasSuffix(c.Request.URL.Path, "account") {
			firstAccountIssued, err := IsFirstAccountIssued()
			if err != nil {
				logger.AuthLog.Errorln(err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authorize user account creation"})
				c.Abort()
				return
			}
			if !firstAccountIssued {
				c.Next()
				return
			}
		}
		claims, err := getClaimsFromAuthorizationHeader(c.Request.Header.Get("Authorization"), jwtSecret)
		if err != nil {
			logger.AuthLog.Errorln(err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("auth failed: %s", err.Error())})
			c.Abort()
			return
		}
		if claims.Permissions == USER_ACCOUNT && strings.HasPrefix(c.Request.URL.Path, "/config/v1/account") {
			requestAllowed, err := isRequestAllowForRegularUser(claims, c.Request.Method, c.Request.URL.Path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authorize operation"})
				c.Abort()
				return
			}
			if !requestAllowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				c.Abort()
				return
			}
		}
		c.Next()
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

func isRequestAllowForRegularUser(claims *jwtGocertClaims, method, path string) (bool, error) {
	allowedPaths := []struct {
		method, pathRegex string
	}{
		{"GET", `/config/v1/account\/(\w+)$`},
		{"POST", `/config/v1/account\/(\w+)\/change_password$`},
	}
	for _, pr := range allowedPaths {
		regex, err := regexp.Compile(pr.pathRegex)
		if err != nil {
			return false, fmt.Errorf("regex couldn't compile: %s", err)
		}
		matches := regex.FindStringSubmatch(path)
		if len(matches) > 0 && method == pr.method {
			if matches[1] == claims.Username {
				return true, nil
			}
			return false, nil
		}
	}
	return false, nil
}
