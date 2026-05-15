// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package webui_context

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/dbadapter"
)

var webuiContext = WEBUIContext{}

type WEBUIContext struct {
	NFProfiles     []models.NFProfile
	NFOamInstances []NfOamInstance
}

type NfOamInstance struct {
	NfId   string
	NfType models.NFType
	Uri    string
}

func init() {
}

func (context *WEBUIContext) UpdateNfProfiles() {
	nfProfilesRaw, errGetMany := dbadapter.CommonDBClient.RestfulAPIGetMany("NfProfile", nil)
	if errGetMany != nil {
		logger.DbLog.Warnln(errGetMany)
	}
	nfProfiles, err := decode(nfProfilesRaw, time.RFC3339)
	if err != nil {
		logger.ContextLog.Error(err)
		return
	}

	context.NFProfiles = nfProfiles

	for _, nfProfile := range context.NFProfiles {
		if nfProfile.NfServices == nil || context.NfProfileAlreadyExists(nfProfile) {
			continue
		}

		var uri string
		switch nfProfile.NfType {
		case models.NFTYPE_AMF:
			uri = getNfOamUri(nfProfile, models.ServiceName("namf-oam"))
		case models.NFTYPE_SMF:
			uri = getNfOamUri(nfProfile, models.ServiceName("nsmf-oam"))
		}
		if uri != "" {
			context.NFOamInstances = append(context.NFOamInstances, NfOamInstance{
				NfId:   nfProfile.NfInstanceId,
				NfType: nfProfile.NfType,
				Uri:    uri,
			})
		}
	}
}

func (context *WEBUIContext) NfProfileAlreadyExists(nfProfile models.NFProfile) bool {
	for _, instance := range context.NFOamInstances {
		if instance.NfId == nfProfile.NfInstanceId {
			return true
		}
	}
	return false
}

func getNfOamUri(nfProfile models.NFProfile, serviceName models.ServiceName) (nfOamUri string) {
	for _, service := range nfProfile.NfServices {
		if service.ServiceName == serviceName && service.NfServiceStatus == models.NFSERVICESTATUS_REGISTERED {
			if nfProfile.GetFqdn() != "" {
				nfOamUri = nfProfile.GetFqdn()
			} else if service.GetFqdn() != "" {
				nfOamUri = service.GetFqdn()
			} else if service.GetApiPrefix() != "" {
				nfOamUri = service.GetApiPrefix()
			} else if len(service.IpEndPoints) > 0 {
				point := (service.IpEndPoints)[0]
				if point.GetIpv4Address() != "" {
					nfOamUri = getSbiUri(service.Scheme, point.GetIpv4Address(), point.GetPort())
				} else if len(nfProfile.Ipv4Addresses) != 0 {
					nfOamUri = getSbiUri(service.Scheme, nfProfile.Ipv4Addresses[0], point.GetPort())
				}
			}
		}
		if nfOamUri != "" {
			break
		}
	}
	return
}

func (context *WEBUIContext) GetOamUris(targetNfType models.NFType) (uris []string) {
	for _, oamInstance := range context.NFOamInstances {
		if oamInstance.NfType == targetNfType {
			uris = append(uris, oamInstance.Uri)
			break
		}
	}
	return
}

func WEBUI_Self() *WEBUIContext {
	return &webuiContext
}

func decode(source interface{}, format string) ([]models.NFProfile, error) {
	var target []models.NFProfile

	// config mapstruct
	stringToDateTimeHook := func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(time.Time{}) && f == reflect.TypeOf("") {
			return time.Parse(format, data.(string))
		}
		return data, nil
	}

	config := mapstructure.DecoderConfig{
		DecodeHook: stringToDateTimeHook,
		Result:     &target,
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return nil, err
	}

	// Decode result to NfProfile structure
	err = decoder.Decode(source)
	if err != nil {
		return nil, err
	}
	return target, nil
}

func getSbiUri(scheme models.UriScheme, ipv4Address string, port int32) (uri string) {
	if port != 0 {
		uri = fmt.Sprintf("%s://%s:%d", scheme, ipv4Address, port)
	} else {
		switch scheme {
		case models.URISCHEME_HTTP:
			uri = fmt.Sprintf("%s://%s:80", scheme, ipv4Address)
		case models.URISCHEME_HTTPS:
			uri = fmt.Sprintf("%s://%s:443", scheme, ipv4Address)
		}
	}
	return
}
