package ssmapi

import (
	"errors"

	ssm_constants "github.com/networkgcorefullcode/ssm/const"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
)

type SSMHSM_API struct{}

var Ssmhsm_api *SSMHSM_API = &SSMHSM_API{}

func (hsm *SSMHSM_API) StoreKey(k4Data *configmodels.K4) error {
	// Implementation for storing key in HSM
	// Check the K4 label keys (AES, DES or DES3)
	if !IsValidKeyIdentifier(k4Data.K4_Label, ssm_constants.KeyLabelsExternalAllow[:]) {
		logger.AppLog.Error("failed to store k4 key in SSM the label key is not valid")
		return errors.New("failed to store k4 key in SSM must key label is incorrect")
	}
	// Check the K4 type to specified the key type that will be store
	if !IsValidKeyIdentifier(k4Data.K4_Type, ssm_constants.KeyTypeAllow[:]) {
		logger.AppLog.Error("failed to store k4 key in SSM the type key is not valid")
		return errors.New("failed to store k4 key in SSM must key type is incorrect")
	}
	// Send the request to the SSM
	resp, err := StoreKeySSM(k4Data.K4_Label, k4Data.K4, k4Data.K4_Type, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to store k4 key in SSM: %+v", err)
		return errors.New("failed to store k4 key in SSM")
	}
	// Check if in the response CipherKey is fill, if it is empty K4 must be a empty string ""
	if resp.CipherKey != "" {
		k4Data.K4 = resp.CipherKey
	} else {
		k4Data.K4 = ""
	}

	return nil
}

func (hsm *SSMHSM_API) UpdateKey(k4Data *configmodels.K4) error {
	// Implementation for updating key in HSM
	// Check the K4 label keys (AES, DES or DES3)
	if !IsValidKeyIdentifier(k4Data.K4_Label, ssm_constants.KeyLabelsExternalAllow[:]) {
		logger.AppLog.Error("failed to update k4 key in SSM the label key is not valid")
		return errors.New("failed to update k4 key in SSM must key label is incorrect")
	}
	// Check the K4 type to specified the key type that will be update
	if !IsValidKeyIdentifier(k4Data.K4_Type, ssm_constants.KeyTypeAllow[:]) {
		logger.AppLog.Error("failed to update k4 key in SSM the type key is not valid")
		return errors.New("failed to update k4 key in SSM must key type is incorrect")
	}
	// Send the request to the SSM
	resp, err := UpdateKeySSM(k4Data.K4_Label, k4Data.K4, k4Data.K4_Type, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to update k4 key in SSM: %+v", err)
		return errors.New("failed to update k4 key in SSM")
	}
	// Check if in the response CipherKey is fill, if it is empty K4 must be a empty string ""
	if resp.CipherKey != "" {
		k4Data.K4 = resp.CipherKey
	} else {
		k4Data.K4 = ""
	}

	return nil
}

func (hsm *SSMHSM_API) DeleteKey(k4Data *configmodels.K4) error {
	// Implementation for deleting key from HSM
	// Check the K4 label keys (both external and internal labels are allowed for deletion)
	if !IsValidKeyIdentifier(k4Data.K4_Label, ssm_constants.KeyLabelsExternalAllow[:]) && !IsValidKeyIdentifier(k4Data.K4_Label, ssm_constants.KeyLabelsInternalAllow[:]) {
		logger.AppLog.Error("failed to delete k4 key in SSM the label key is not valid")
		return errors.New("failed to delete k4 key in SSM must key label is incorrect")
	}
	// Send the request to the SSM
	_, err := DeleteKeySSM(k4Data.K4_Label, int32(k4Data.K4_SNO))
	if err != nil {
		logger.AppLog.Errorf("failed to delete k4 key in SSM: %+v", err)
		return errors.New("failed to delete k4 key in SSM")
	}

	return nil
}
