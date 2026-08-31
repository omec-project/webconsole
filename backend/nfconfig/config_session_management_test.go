// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
//

package nfconfig

import (
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

const (
	sliceNameSlice1    = "slice-1"
	sliceNameSlice4    = "slice-4"
	upfHostnameLocal   = "upf.local"
	gnbNameGnb1        = "gnb-1"
	deviceGroupNameDG9 = "dg-9"
	upfHostnameB       = "upf-b.local"
	dnsPrimaryAlt      = "1.1.1.1"
	upfHostnameCom     = "hostname.com"
)

type networkSliceParams struct {
	sliceName    string
	mcc          string
	mnc          string
	sst          string
	sd           string
	deviceGroups []string
	upfHostname  any
	upfPort      any
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
					sliceName:    sliceNameSlice1,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{deviceGroupNameDG1},
					upfHostname:  upfHostnameLocal,
					upfPort:      "8805",
					gnbNames:     []string{gnbNameGnb1},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:         deviceGroupNameDG1,
					dnn:          dnnInternet,
					dnsPrimary:   dnsPrimaryTest,
					pcscfPrimary: "10.10.10.10",
					ueIpPool:     ueIpPoolTest,
					mtu:          1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice1,
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
							DnnName:   dnnInternet,
							DnsIpv4:   dnsPrimaryTest,
							PcscfIpv4: openapi.PtrString("10.10.10.10"),
							UeSubnet:  ueIpPoolTest,
							Mtu:       1500,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameLocal,
						Port:     ptr(int32(8805)),
					},
					GnbNames: []string{gnbNameGnb1},
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
					sliceName:    sliceNameSlice1,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{deviceGroupNameDG1},
					gnbNames:     []string{gnbNameGnb1},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice1,
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
							DnnName:  dnnInternet,
							DnsIpv4:  dnsPrimaryTest,
							UeSubnet: ueIpPoolTest,
							Mtu:      1500,
						},
					},
					GnbNames: []string{gnbNameGnb1},
				},
			},
		},
		{
			name: "Slice missing device group",
			sliceParams: []networkSliceParams{
				{
					sliceName:   sliceNameSlice1,
					mcc:         "001",
					mnc:         "01",
					sst:         "1",
					sd:          "010203",
					upfHostname: upfHostnameLocal,
					upfPort:     "8805",
					gnbNames:    []string{gnbNameGnb1},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
					dnsPrimary: dnsPrimaryTest,
					ueIpPool:   ueIpPoolTest,
					mtu:        1500,
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice1,
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameLocal,
						Port:     ptr(int32(8805)),
					},
					GnbNames: []string{gnbNameGnb1},
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
					deviceGroups: []string{deviceGroupNameDG9},
					upfHostname:  upfHostnameB,
				},
				{
					sliceName:    "slice-e",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{deviceGroupNameDG9},
					upfHostname:  upfHostnameB,
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG9,
					dnn:        dnnInternet,
					dnsPrimary: dnsPrimaryAlt,
					ueIpPool:   ueIpPoolTest,
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
							DnnName:  dnnInternet,
							DnsIpv4:  dnsPrimaryAlt,
							UeSubnet: ueIpPoolTest,
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameB,
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
							DnnName:  dnnInternet,
							DnsIpv4:  dnsPrimaryAlt,
							UeSubnet: ueIpPoolTest,
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameB,
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
					upfHostname:  upfHostnameLocal,
					upfPort:      "invalid",
					deviceGroups: []string{deviceGroupNameDG1},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
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
							DnnName:  dnnInternet,
							DnsIpv4:  "9.9.9.9",
							UeSubnet: "10.2.2.0/24",
							Mtu:      1500,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameLocal,
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
					upfHostname:  upfHostnameLocal,
					upfPort:      "2152",
					deviceGroups: []string{deviceGroupNameDG1},
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       deviceGroupNameDG1,
					dnn:        dnnInternet,
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
							DnnName:  dnnInternet,
							DnsIpv4:  "4.4.4.4",
							UeSubnet: "10.3.3.0/24",
							Mtu:      1400,
						},
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameLocal,
						Port:     ptr(int32(2152)),
					},
				},
			},
		},
		{
			name: "invalid upf hostname (non-string)",
			sliceParams: []networkSliceParams{
				{
					sliceName:    sliceNameSlice4,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  1234,
					deviceGroups: []string{deviceGroupNameDG1},
					gnbNames:     []string{gnbNameGnb1},
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice4,
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf:      nil,
					GnbNames: []string{gnbNameGnb1},
				},
			},
		},
		{
			name: "int upf port is valid",
			sliceParams: []networkSliceParams{
				{
					sliceName:    sliceNameSlice4,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  upfHostnameCom,
					upfPort:      5677,
					deviceGroups: []string{deviceGroupNameDG1},
					gnbNames:     []string{gnbNameGnb1},
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice4,
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameCom,
						Port:     ptr(int32(5677)),
					},
					GnbNames: []string{gnbNameGnb1},
				},
			},
		},
		{
			name: "float upf port is valid",
			sliceParams: []networkSliceParams{
				{
					sliceName:    sliceNameSlice4,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					upfHostname:  upfHostnameCom,
					upfPort:      1234.00,
					deviceGroups: []string{deviceGroupNameDG1},
					gnbNames:     []string{gnbNameGnb1},
				},
			},
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice4,
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameCom,
						Port:     ptr(int32(1234)),
					},
					GnbNames: []string{gnbNameGnb1},
				},
			},
		},
		{
			name: "empty device group list",
			sliceParams: []networkSliceParams{
				{
					sliceName:    sliceNameSlice1,
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "010203",
					deviceGroups: []string{},
					upfHostname:  upfHostnameLocal,
				},
			},
			deviceGroups: nil,
			expectedResponse: []nfConfigApi.SessionManagement{
				{
					SliceName: sliceNameSlice1,
					PlmnId: nfConfigApi.PlmnId{
						Mcc: "001",
						Mnc: "01",
					},
					Snssai: nfConfigApi.Snssai{
						Sst: 1,
						Sd:  sharedSd,
					},
					Upf: &nfConfigApi.Upf{
						Hostname: upfHostnameLocal,
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
