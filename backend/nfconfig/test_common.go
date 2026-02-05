// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

type deviceGroupParams struct {
	name       string
	imsis      []string
	dnn        string
	dnsPrimary string
	ueIpPool   string
	mtu        int32
	qos        *configmodels.DeviceGroupsIpDomainExpandedUeDnnQos
}

func makeDeviceGroup(p deviceGroupParams) (string, configmodels.DeviceGroups) {
	return p.name, configmodels.DeviceGroups{
		Imsis: p.imsis,
		IpDomainExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			{
				Dnn:        p.dnn,
				DnsPrimary: p.dnsPrimary,
				UeIpPool:   p.ueIpPool,
				Mtu:        p.mtu,
				UeDnnQos:   p.qos,
			},
		},
	}
}

func makeSnssaiWithSd(sst int32, sd string) nfConfigApi.Snssai {
	s := nfConfigApi.NewSnssai(sst)
	s.SetSd(sd)
	return *s
}
