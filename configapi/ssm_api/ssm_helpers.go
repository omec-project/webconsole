package ssmapi

import (
	"slices"

	ssm "github.com/networkgcorefullcode/ssm/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/backend/ssm/apiclient"
)

func StoreKeySSM(keyLabel, keyValue, keyType string, keyID int32) (*ssm.StoreKeyResponse, error) {
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", keyLabel, keyID, keyType)
	storeKeyRequest := ssm.StoreKeyRequest{
		KeyLabel: keyLabel,
		Id:       keyID,
		KeyValue: keyValue,
		KeyType:  keyType,
	}
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", storeKeyRequest.KeyLabel, storeKeyRequest.Id, storeKeyRequest.KeyType)

	apiClient := apiclient.GetSSMAPIClient()

	resp, r, err := apiClient.KeyManagementAPI.StoreKey(apiclient.AuthContext).StoreKeyRequest(storeKeyRequest).Execute()
	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.StoreKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return nil, err
	}
	logger.WebUILog.Infof("Response from `KeyManagementAPI.StoreKey`: %+v", resp)
	return resp, nil
}

func UpdateKeySSM(keyLabel, keyValue, keyType string, keyID int32) (*ssm.UpdateKeyResponse, error) {
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", keyLabel, keyID, keyType)
	updateKeyRequest := ssm.UpdateKeyRequest{
		KeyLabel: keyLabel,
		Id:       keyID,
		KeyValue: keyValue,
		KeyType:  keyType,
	}
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", updateKeyRequest.KeyLabel, updateKeyRequest.Id, updateKeyRequest.KeyType)

	apiClient := apiclient.GetSSMAPIClient()

	resp, r, err := apiClient.KeyManagementAPI.UpdateKey(apiclient.AuthContext).UpdateKeyRequest(updateKeyRequest).Execute()
	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.StoreKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return nil, err
	}
	logger.WebUILog.Infof("Response from `KeyManagementAPI.StoreKey`: %+v", resp)
	return resp, nil
}

func DeleteKeySSM(keyLabel string, keyID int32) (*ssm.DeleteKeyResponse, error) {
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", keyLabel, keyID)
	deleteKeyRequest := ssm.DeleteKeyRequest{
		KeyLabel: keyLabel,
		Id:       keyID,
	}
	logger.AppLog.Debugf("key label: %s key id: %s key type: %s", deleteKeyRequest.KeyLabel, deleteKeyRequest.Id)

	apiClient := apiclient.GetSSMAPIClient()

	resp, r, err := apiClient.KeyManagementAPI.DeleteKey(apiclient.AuthContext).DeleteKeyRequest(deleteKeyRequest).Execute()
	if err != nil {
		logger.AppLog.Errorf("Error when calling `KeyManagementAPI.StoreKey`: %v", err)
		logger.AppLog.Errorf("Full HTTP response: %v", r)
		return nil, err
	}
	logger.WebUILog.Infof("Response from `KeyManagementAPI.StoreKey`: %+v", resp)
	return resp, nil
}

func IsValidKeyIdentifier(keyLabel string, keyIdentifier []string) bool {
	if keyLabel == "" {
		return false
	}
	return slices.Contains(keyIdentifier, keyLabel)
}
