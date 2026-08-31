// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

const (
	testSiteName                   = "test"
	testSliceName1                 = "slice1"
	nameSeveralSlicesDifferentPLMN = "Several slices different PLMN are ordered"
	nameEmptySlices                = "Empty slices"
	deviceGroupNameDG1             = "dg-1"
	dnnInternet                    = "internet"
	imsiTest                       = "001010123456789"
	dnsPrimaryTest                 = "8.8.8.8"
	ueIpPoolTest                   = "10.1.1.0/24"
)

type deviceGroupParams struct {
	name         string
	imsis        []string
	dnn          string
	dnsPrimary   string
	pcscfPrimary string
	ueIpPool     string
	mtu          int32
	qos          *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos
}

func makeDeviceGroup(p deviceGroupParams) (string, configmodels.DeviceGroups) {
	return p.name, configmodels.DeviceGroups{
		Imsis: p.imsis,
		IpDomainsExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			{
				Dnn:          p.dnn,
				DnsPrimary:   p.dnsPrimary,
				PcscfPrimary: p.pcscfPrimary,
				UeIpPool:     p.ueIpPool,
				Mtu:          p.mtu,
				UeDnnQos:     p.qos,
			},
		},
	}
}

func makeSnssaiWithSd(sst int32, sd string) nfConfigApi.Snssai {
	s := nfConfigApi.NewSnssai(sst)
	s.SetSd(sd)
	return *s
}
