package utils

import (
	"errors"
	"os"

	"github.com/omec-project/webconsole/backend/factory"
)

// GetUserLogin retrieves the SSM service ID and password from configuration or environment variables
func GetUserLogin() (string, string, error) {
	var username, password string

	if factory.WebUIConfig.Configuration.SSM.Login != nil {
		username = factory.WebUIConfig.Configuration.SSM.Login.ServiceId
		password = factory.WebUIConfig.Configuration.SSM.Login.Password
	} else {
		username = os.Getenv("SSM_SERVICE_ID")
		password = os.Getenv("SSM_PASSWORD")
	}

	if username == "" || password == "" {
		return "", "", errors.New("SSM login credentials are not set")
	}
	return username, password, nil
}
