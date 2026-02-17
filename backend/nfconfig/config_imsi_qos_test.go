// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package nfconfig

import (
	"reflect"
	"testing"

	"github.com/omec-project/openapi/nfConfigApi"
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
					name:       "dg-1",
					dnn:        "internet",
					imsis:      []string{"001010123456789"},
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
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
					imsis: []string{"001010123456789"},
					dnn:   "internet",
					qos: []nfConfigApi.ImsiQos{
						*nfConfigApi.NewImsiQos("20 Mbps", "200 Mbps", 6, 9),
					},
				},
			},
		},
		{
			name: "Multiple DeviceGroups produce multiple imsiQosConfigs",
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					imsis:      []string{"001010123456789"},
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
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
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
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
					imsis: []string{"001010123456789"},
					dnn:   "internet",
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
