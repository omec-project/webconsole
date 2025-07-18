// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"reflect"
	"testing"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

type networkSliceParams struct {
	sliceName    string
	mcc          string
	mnc          string
	sst          string
	sd           string
	deviceGroups []string
	upfHostname  any
	upfPort      string
	gnbNames     []string
}

func prepareNetworkSlice(p networkSliceParams) configmodels.Slice {
	upf := map[string]any{
		"upf-name": p.upfHostname,
	}
	if p.upfPort != "" {
		upf["upf-port"] = p.upfPort
	}

	var gnbs []configmodels.SliceSiteInfoGNodeBs
	for _, name := range p.gnbNames {
		gnbs = append(gnbs, configmodels.SliceSiteInfoGNodeBs{
			Name: name,
			Tac:  1,
		})
	}

	return configmodels.Slice{
		SliceName: p.sliceName,
		SliceId: configmodels.SliceSliceId{
			Sst: p.sst,
			Sd:  p.sd,
		},
		SiteDeviceGroup: p.deviceGroups,
		SiteInfo: configmodels.SliceSiteInfo{
			SiteName: "demo",
			Plmn: configmodels.SliceSiteInfoPlmn{
				Mcc: p.mcc,
				Mnc: p.mnc,
			},
			GNodeBs: gnbs,
			Upf:     upf,
		},
	}
}

func prepareMultipleSlices(params []networkSliceParams) []configmodels.Slice {
	var slices []configmodels.Slice
	for _, p := range params {
		slices = append(slices, prepareNetworkSlice(p))
	}
	return slices
}

func ptr[T any](v T) *T {
	return &v
}

var sharedSd = ptr("010203")

func TestSyncSessionManagement(t *testing.T) {
	tests := []struct {
		name             string
		sliceParams      []networkSliceParams
		deviceGroups     []deviceGroupParams
		expectedError    bool
		expectedResponse []nfConfigApi.SessionManagement
	}{
		{
			name: "valid slice with all fields",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-1",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{"dg-1"},
					upfHostname:  "upf.local",
					upfPort:      "8805",
					gnbNames:     []string{"gnb-1"},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-1",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "8.8.8.8",
							UeSubnet: "10.1.1.0/24",
							Mtu:      1500,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf.local",
						Port:     ptr(int32(8805)),
					},
					GnbNames: []string{"gnb-1"},
				},
			},
		},
		{
			name: "invalid SST",
			sliceParams: []networkSliceParams{
				{
					sliceName: "bad-slice",
					mcc:       "001",
					mnc:       "01",
					sst:       "",
					sd:        "010203",
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{},
		},
		{
			name: "Slice missing UPF",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-1",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{"dg-1"},
					gnbNames:     []string{"gnb-1"},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-1",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "8.8.8.8",
							UeSubnet: "10.1.1.0/24",
							Mtu:      1500,
						},
					},
					GnbNames: []string{"gnb-1"},
				},
			},
		},
		{
			name: "Slice missing device group",
			sliceParams: []networkSliceParams{
				{
					sliceName:   "slice-1",
					mcc:         "001",
					mnc:         "01",
					sst:         "1",
					sd:          "010203",
					upfHostname: "upf.local",
					upfPort:     "8805",
					gnbNames:    []string{"gnb-1"},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					dnsPrimary: "8.8.8.8",
					ueIpPool:   "10.1.1.0/24",
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-1",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf.local",
						Port:     ptr(int32(8805)),
					},
					GnbNames: []string{"gnb-1"},
				},
			},
		},
		{
			name: "multiple slices should be sorted by slice-name",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-f",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{"dg-9"},
					upfHostname:  "upf-b.local",
				},
				{
					sliceName:    "slice-e",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{"dg-9"},
					upfHostname:  "upf-b.local",
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-9",
					dnn:        "internet",
					dnsPrimary: "1.1.1.1",
					ueIpPool:   "10.1.1.0/24",
					mtu:        1400,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-e",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "1.1.1.1",
							UeSubnet: "10.1.1.0/24",
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf-b.local",
					},
				},
				{
					SliceName: "slice-f",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "1.1.1.1",
							UeSubnet: "10.1.1.0/24",
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf-b.local",
					},
				},
			},
		},
		{
			name: "valid upf hostname but invalid port",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-2",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  "upf.local",
					upfPort:      "invalid",
					deviceGroups: []string{"dg-1"},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					dnsPrimary: "9.9.9.9",
					ueIpPool:   "10.2.2.0/24",
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-2",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "9.9.9.9",
							UeSubnet: "10.2.2.0/24",
							Mtu:      1500,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf.local",
					},
				},
			},
		},
		{
			name: "valid upf hostname and port",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-3",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  "upf.local",
					upfPort:      "2152",
					deviceGroups: []string{"dg-1"},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-1",
					dnn:        "internet",
					dnsPrimary: "4.4.4.4",
					ueIpPool:   "10.3.3.0/24",
					mtu:        1400,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-3",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					IpDomain: []nfConfigApi.IpDomain{
						{
							DnnName:  "internet",
							DnsIpv4:  "4.4.4.4",
							UeSubnet: "10.3.3.0/24",
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf.local",
						Port:     ptr(int32(2152)),
					},
				},
			},
		},
		{
			name: "invalid upf hostname (non-string)",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-4",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  1234,
					deviceGroups: []string{"dg-1"},
					gnbNames:     []string{"gnb-1"},
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-4",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf:      nil,
					GnbNames: []string{"gnb-1"},
				},
			},
		},
		{
			name: "empty device group list",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-1",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{},
					upfHostname:  "upf.local",
				},
			},
			deviceGroups: nil,
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: "slice-1",
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: "upf.local",
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

			slices := prepareMultipleSlices(tt.sliceParams)
			cfg := inMemoryConfig{}
			cfg.syncSessionManagement(slices, deviceGroupMap)

			if !reflect.DeepEqual(cfg.sessionManagement, tt.expectedResponse) {
				t.Errorf("expected %+v, got %+v", tt.expectedResponse, cfg.sessionManagement)
			}
		})
	}
}
