package apiclient

import (
	"context"

	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/logger"
)

var (
	AuthContext context.Context = context.Background()
	CurrentJWT  string          = ""
)

// SetAuthContext sets the authentication context with the provided JWT token
func SetAuthContext(jwt string) {
	AuthContext = context.WithValue(context.Background(), ssm_models.ContextAccessToken, jwt)
	CurrentJWT = jwt
}

// LoginSSM performs login to the SSM and returns the authentication token
func LoginSSM(serviceId, password string) (string, error) {
	loginRequest := ssm_models.LoginRequest{
		ServiceId: serviceId,
		Password:  password,
	}

	client := GetSSMAPIClient()

	resp, r, err := client.AuthenticationAPI.UserLogin(context.Background()).LoginRequest(loginRequest).Execute()
	if err != nil {
		logger.WebUILog.Errorf("Error when calling `AuthenticationAPI.UserLogin`: %v", err)
		logger.WebUILog.Errorf("Full HTTP response: %v", r)
		return "", err
	}
	// response from `UserLogin`: LoginResponse
	logger.WebUILog.Infof("Response from `AuthenticationAPI.UserLogin`: %s", resp.Message)
	SetAuthContext(resp.Token)
	return resp.Token, nil
}
