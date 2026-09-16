// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package nfconfig

import (
	"math"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

func TestSyncImsiQos(t *testing.T) {
	tests := []struct {
		name             string
		deviceGroups     []deviceGroupParams
		expectedResponse []imsiQosConfig
	}{
		{
			name: "Single DeviceGroup produces one imsiQosConfig",
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
					imsis:      []string{imsiTest},
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
					qos: &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
						DnnMbrUplink:   20000000,
						DnnMbrDownlink: 200000000,
						TrafficClass: &configmodels.TrafficClassInfo{
							Qci: 6,
							Arp: 9,
						},
					},
				},
			},
			expectedResponse: []imsiQosConfig{
				{
					imsis: []string{imsiTest},
					dnn:   dnnInternet,
					qos: []nfConfigApi.ImsiQos{
						*nfConfigApi.NewImsiQos("20 Mbps", "200 Mbps", 6, 9),
					},
				},
			},
		},
		{
			// This is read straight from storage, bypassing configapi's ingest validation, so a
			// group written under the old clamp still holds math.MaxInt64 here.
			name: "DeviceGroup with a legacy math.MaxInt64 rate is clamped rather than served unreadably",
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
					imsis:      []string{imsiTest},
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
					qos: &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
						DnnMbrUplink:   math.MaxInt64,
						DnnMbrDownlink: math.MaxInt64,
						TrafficClass: &configmodels.TrafficClassInfo{
							Qci: 6,
							Arp: 9,
						},
					},
				},
			},
			expectedResponse: []imsiQosConfig{
				{
					imsis: []string{imsiTest},
					dnn:   dnnInternet,
					qos: []nfConfigApi.ImsiQos{
						*nfConfigApi.NewImsiQos("65535 Gbps", "65535 Gbps", 6, 9),
					},
				},
			},
		},
		{
			name: "Multiple DeviceGroups produce multiple imsiQosConfigs",
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
					imsis:      []string{imsiTest},
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
					qos: &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
						DnnMbrUplink:   20000000,
						DnnMbrDownlink: 200000000,
						TrafficClass: &configmodels.TrafficClassInfo{
							Qci: 6,
							Arp: 9,
						},
					},
				},
				{
					name:       "dg-2",
					dnn:        "connection",
					imsis:      []string{"001010123456790", "001010123456791"},
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
					qos: &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
						DnnMbrUplink:   10000000,
						DnnMbrDownlink: 100000000,
						TrafficClass: &configmodels.TrafficClassInfo{
							Qci: 3,
							Arp: 6,
						},
					},
				},
			},
			expectedResponse: []imsiQosConfig{
				{
					imsis: []string{"001010123456790", "001010123456791"},
					dnn:   "connection",
					qos: []nfConfigApi.ImsiQos{
						*nfConfigApi.NewImsiQos("10 Mbps", "100 Mbps", 3, 6),
					},
				},
				{
					imsis: []string{imsiTest},
					dnn:   dnnInternet,
					qos: []nfConfigApi.ImsiQos{
						*nfConfigApi.NewImsiQos("20 Mbps", "200 Mbps", 6, 9),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceGroupMap := make(map[string]configmodels.DeviceGroups)
			for _, dg := range tt.deviceGroups {
				name, group := makeDeviceGroup(dg)
				deviceGroupMap[name] = group
			}

			cfg := inMemoryConfig{}
			cfg.syncImsiQos(deviceGroupMap)

			if !reflect.DeepEqual(cfg.imsiQos, tt.expectedResponse) {
				t.Errorf("expected %+v, got %+v", tt.expectedResponse, cfg.imsiQos)
			}
		})
	}
}
