package ssmsync

import (
	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	ssm_models "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
	"github.com/omec-project/webconsole/configmodels"
)

type CreateKeySSM interface {
	CreateNewKeySSM(keyLabel string, id int32) (configmodels.K4, error)
}

type CreateAES128SSM struct{}

func (c *CreateAES128SSM) CreateNewKeySSM(keyLabel string, id int32) (configmodels.K4, error) {
	logger.AppLog.Infof("Creating new AES-128 key in SSM with label %s, id %d", keyLabel, id)

	var genAESKeyRequest ssm_models.GenAESKeyRequest = ssm_models.GenAESKeyRequest{
		Id:   id,
		Bits: 128,
	}

	apiClient := apiclient.GetSSMAPIClient()

	_, r, err := apiClient.KeyManagementAPI.GenerateAESKey(apiclient.AuthContext).GenAESKeyRequest(genAESKeyRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateAESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return configmodels.K4{}, err
	}

	return configmodels.K4{
		K4:       "",
		K4_Type:  ssm_constants.TYPE_AES,
		K4_SNO:   byte(id),
		K4_Label: keyLabel,
	}, nil
}

type CreateAES256SSM struct{}

func (c *CreateAES256SSM) CreateNewKeySSM(keyLabel string, id int32) (configmodels.K4, error) {
	logger.AppLog.Infof("Creating new AES-256 key in SSM with label %s, id %d", keyLabel, id)

	var genAESKeyRequest ssm_models.GenAESKeyRequest = ssm_models.GenAESKeyRequest{
		Id:   id,
		Bits: 256,
	}

	apiClient := apiclient.GetSSMAPIClient()

	_, r, err := apiClient.KeyManagementAPI.GenerateAESKey(apiclient.AuthContext).GenAESKeyRequest(genAESKeyRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateAESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return configmodels.K4{}, err
	}

	return configmodels.K4{
		K4:       "",
		K4_Type:  ssm_constants.TYPE_AES,
		K4_SNO:   byte(id),
		K4_Label: keyLabel,
	}, nil
}

type CreateDes3SSM struct{}

func (c *CreateDes3SSM) CreateNewKeySSM(keyLabel string, id int32) (configmodels.K4, error) {
	logger.AppLog.Infof("Creating new DES3 key in SSM with label %s, id %d", keyLabel, id)

	var genDES3KeyRequest ssm_models.GenDES3KeyRequest = ssm_models.GenDES3KeyRequest{
		Id: id,
	}

	apiClient := apiclient.GetSSMAPIClient()
	_, r, err := apiClient.KeyManagementAPI.GenerateDES3Key(apiclient.AuthContext).GenDES3KeyRequest(genDES3KeyRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateDES3Key`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return configmodels.K4{}, err
	}

	return configmodels.K4{
		K4:       "",
		K4_Type:  ssm_constants.TYPE_DES3,
		K4_SNO:   byte(id),
		K4_Label: keyLabel,
	}, nil
}

type CreateDesSSM struct{}

func (c *CreateDesSSM) CreateNewKeySSM(keyLabel string, id int32) (configmodels.K4, error) {
	logger.AppLog.Infof("Creating new DES key in SSM with label %s, id %d", keyLabel, id)

	var genDESKeyRequest ssm_models.GenDESKeyRequest = ssm_models.GenDESKeyRequest{
		Id: id,
	}

	apiClient := apiclient.GetSSMAPIClient()
	_, r, err := apiClient.KeyManagementAPI.GenerateDESKey(apiclient.AuthContext).GenDESKeyRequest(genDESKeyRequest).Execute()

	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.GenerateDESKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return configmodels.K4{}, err
	}

	return configmodels.K4{
		K4:       "",
		K4_Type:  ssm_constants.TYPE_DES,
		K4_SNO:   byte(id),
		K4_Label: keyLabel,
	}, nil
}
