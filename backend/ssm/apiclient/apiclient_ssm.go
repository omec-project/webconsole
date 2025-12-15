package apiclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
)

var apiClient *ssm_models.APIClient

// GetSSMAPIClient creates and returns a configured SSM API client
func GetSSMAPIClient() *ssm_models.APIClient {
	if apiClient != nil {
		logger.AppLog.Debugf("Returning existing SSM API client")
		return apiClient
	}

	logger.AppLog.Infof("Creating new SSM API client for URI: %s", factory.WebUIConfig.Configuration.SSM.SsmUri)

	configuration := ssm_models.NewConfiguration()
	configuration.Servers[0].URL = factory.WebUIConfig.Configuration.SSM.SsmUri
	configuration.HTTPClient = GetHTTPClient(factory.WebUIConfig.Configuration.SSM.TLS_Insecure)

	if factory.WebUIConfig.Configuration.SSM.MTls != nil {
		logger.AppLog.Infof("Configuring mTLS for SSM client")

		// 1️⃣ Load client certificate for mTLS
		logger.AppLog.Debugf("Loading client certificate from: %s", factory.WebUIConfig.Configuration.SSM.MTls.Crt)
		cert, err := tls.LoadX509KeyPair(factory.WebUIConfig.Configuration.SSM.MTls.Crt, factory.WebUIConfig.Configuration.SSM.MTls.Key)
		if err != nil {
			logger.AppLog.Errorf("Error loading client certificate: %v", err)
			fmt.Fprintf(os.Stderr, "Error loading client certificate: %v\n", err)
			return nil
		}
		logger.AppLog.Infof("Client certificate loaded successfully")

		// 2️⃣ Load root certificate (CA) that signed the server
		logger.AppLog.Debugf("Loading CA certificate from: %s", factory.WebUIConfig.Configuration.SSM.MTls.Ca)
		caCert, err := os.ReadFile(factory.WebUIConfig.Configuration.SSM.MTls.Ca)
		if err != nil {
			logger.AppLog.Errorf("Error reading CA certificate: %v", err)
			fmt.Fprintf(os.Stderr, "Error reading CA: %v\n", err)
			return nil
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		logger.AppLog.Infof("CA certificate loaded successfully")

		// 3️⃣ Configure TLS
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert}, // client authentication
			RootCAs:      caCertPool,              // verify server
			MinVersion:   tls.VersionTLS12,
		}
		logger.AppLog.Debugf("TLS configuration created with MinVersion: TLS 1.2")

		// 4️⃣ Create an HTTP client with this configuration
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		httpClient := &http.Client{Transport: transport}

		if factory.WebUIConfig.Configuration.SSM.TLS_Insecure {
			logger.AppLog.Warnf("TLS_Insecure enabled - skipping certificate verification")
			httpClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
		}

		// 5️⃣ Configure the OpenAPI client to use this HTTP client
		configuration.HTTPClient = httpClient
		logger.AppLog.Infof("mTLS HTTP client configured successfully")
	} else {
		logger.AppLog.Infof("mTLS not configured, using default HTTP client")
	}

	apiClient = ssm_models.NewAPIClient(configuration)
	logger.AppLog.Infof("SSM API client created successfully")

	return apiClient
}

// getHTTPClient returns an HTTP client configured based on TLS settings
func GetHTTPClient(tlsInsecure bool) *http.Client {
	if tlsInsecure {
		// Create client with insecure TLS configuration
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	// Return default HTTP client for secure connections
	return &http.Client{}
}
