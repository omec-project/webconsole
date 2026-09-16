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

// A device group written before its rates were bounded holds the math.MaxInt64 that the old ingest
// path produced from a negative rate. Nothing relabels or repairs the stored row on this path --
// the loader unmarshals it straight from the database and never reads its unit -- so the rate has
// to be brought into range where it is rendered, or it is served as "9223372036854775807 bps": a
// numeral nas's Session-AMBR converter cannot parse and smf's GetBitRate reads as Mbps through an
// implementation-defined uint16 conversion.
func TestSyncImsiQosBoundsARateWrittenBeforeTheyWereBounded(t *testing.T) {
	name, group := makeDeviceGroup(deviceGroupParams{
		name:       deviceGroupNameDG1,
		dnn:        dnnInternet,
		imsis:      []string{imsiTest},
		dnsPrimary: dnsPrimaryTest,
		ueIpPool:   ueIpPoolTest,
		mtu:        1500,
		qos: &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{
			DnnMbrUplink:   math.MaxInt64,
			DnnMbrDownlink: 20000000,
			TrafficClass:   &configmodels.TrafficClassInfo{Qci: 6, Arp: 9},
		},
	})

	cfg := inMemoryConfig{}
	cfg.syncImsiQos(map[string]configmodels.DeviceGroups{name: group})

	if len(cfg.imsiQos) != 1 || len(cfg.imsiQos[0].qos) != 1 {
		t.Fatalf("expected one IMSI QoS entry, got %+v", cfg.imsiQos)
	}
	served := cfg.imsiQos[0].qos[0]
	if served.GetMbrUplink() != "65535 Gbps" {
		t.Errorf("mbrUplink = %q, want %q", served.GetMbrUplink(), "65535 Gbps")
	}
	// The rate that was always in range is untouched, so the bound is not a blanket rewrite.
	if served.GetMbrDownlink() != "20 Mbps" {
		t.Errorf("mbrDownlink = %q, want %q", served.GetMbrDownlink(), "20 Mbps")
	}
}
