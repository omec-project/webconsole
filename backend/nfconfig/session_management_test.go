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
	upfHostname  string
	upfPort      string
	gnbNames     []string
}

func prepareNetworkSlice(p networkSliceParams) configmodels.Slice {
	upf := map[string]interface{}{
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

type deviceGroupParams struct {
	name       string
	dnn        string
	dnsPrimary string
	ueIpPool   string
	mtu        int32
}

func makeDeviceGroup(p deviceGroupParams) (string, configmodels.DeviceGroups) {
	return p.name, configmodels.DeviceGroups{
		IpDomainExpanded: configmodels.DeviceGroupsIpDomainExpanded{
			Dnn:        p.dnn,
			DnsPrimary: p.dnsPrimary,
			UeIpPool:   p.ueIpPool,
			Mtu:        p.mtu,
		},
	}
}

func TestSyncSessionManagement(t *testing.T) {
	tests := []struct {
		name          string
		sliceParams   []networkSliceParams
		deviceGroups  []deviceGroupParams
		expectedError bool
		validateFunc  func(*testing.T, []nfConfigApi.SessionManagement)
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				s := sessions[0]
				if s.GetSliceName() != "slice-1" {
					t.Errorf("expected slice name 'slice-1', got %s", s.GetSliceName())
				}

				expectedPlmn := nfConfigApi.PlmnId{
					Mcc: "001",
					Mnc: "01",
				}

				if got := s.GetPlmnId(); got != expectedPlmn {
					t.Errorf("unexpected PLMN: got %+v, expected %+v", got, expectedPlmn)
				}

				sd := "010203"
				expectedSnssai := nfConfigApi.Snssai{
					Sst: 1,
					Sd:  &sd,
				}
				if got := s.GetSnssai(); !reflect.DeepEqual(got, expectedSnssai) {
					t.Errorf("unexpected SNSSAI: got %+v, expected %+v", got, expectedSnssai)
				}

				expectedUpf := nfConfigApi.Upf{Hostname: "upf.local"}
				expectedUpf.SetPort(8805)
				gotUpf := s.GetUpf()

				if gotUpf.GetHostname() != expectedUpf.GetHostname() || gotUpf.GetPort() != expectedUpf.GetPort() {
					t.Errorf("unexpected UPF: got %+v, expected %+v", gotUpf, expectedUpf)
				}
				expectedIpDomain := nfConfigApi.IpDomain{
					DnnName:  "internet",
					DnsIpv4:  "8.8.8.8",
					UeSubnet: "10.1.1.0/24",
					Mtu:      1500,
				}
				if got := s.GetIpDomain(); len(got) == 0 || got[0] != expectedIpDomain {
					t.Errorf("unexpected IP domain: got %+v, expected %+v", got, []nfConfigApi.IpDomain{expectedIpDomain})
				}

				if got := s.GetGnbNames(); !reflect.DeepEqual(got, []string{"gnb-1"}) {
					t.Errorf("unexpected gNBs: got %+v, expected %+v", got, []string{"gnb-1"})
				}
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 0 {
					t.Errorf("expected no sessions with invalid SST, got %d", len(sessions))
				}
			},
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				s := sessions[0]
				if s.Upf.GetHostname() != "" {
					t.Errorf("expected UPF not available, got %+v", s.GetUpf())
				}
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
					upfHostname: "12123",
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				s := sessions[0]
				if len(s.GetIpDomain()) != 0 {
					t.Errorf("expected no IP domains due to missing device group, got %+v", s.GetIpDomain())
				}
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
					sd:           "000001",
					deviceGroups: []string{"dg-9"},
					upfHostname:  "upf-b.local",
				},
				{
					sliceName:    "slice-e",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "000002",
					deviceGroups: []string{"dg-9"},
					upfHostname:  "upf-b.local",
				},
			},
			deviceGroups: []deviceGroupParams{
				{
					name:       "dg-9",
					dnn:        "internet",
					dnsPrimary: "1.1.1.1",
					ueIpPool:   "10.1.2.0/24",
					mtu:        1400,
				},
			},
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 2 {
					t.Fatalf("expected 2 session, got %d", len(sessions))
				}
				if sessions[0].GetSliceName() != "slice-e" {
					t.Errorf("expected slice to be 'slice-e', got %s", sessions[0].GetSliceName())
				}
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				if sessions[0].Upf.GetHostname() != "upf.local" {
					t.Errorf("expected UPF hostname 'upf.local', got %s", sessions[0].Upf.GetHostname())
				}
				if sessions[0].Upf.GetPort() != 0 {
					t.Errorf("expected UPF port to be 0 due to invalid input, got %d", sessions[0].Upf.GetPort())
				}
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
					sd:           "030201",
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
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				upf := sessions[0].GetUpf()
				if upf.GetHostname() != "upf.local" {
					t.Errorf("expected hostname upf.local, got %s", upf.GetHostname())
				}
				if upf.GetPort() != 2152 {
					t.Errorf("expected port 2152, got %d", upf.GetPort())
				}
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
					sd:           "040302",
					deviceGroups: []string{"dg-1"},
					gnbNames:     []string{"gnb-1"},
				},
			},
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				if sessions[0].Upf.GetHostname() != "" {
					t.Errorf("expected empty hostname due to invalid type, got: %s", sessions[0].Upf.GetHostname())
				}
			},
		},
		{
			name: "empty device group list",
			sliceParams: []networkSliceParams{
				{
					sliceName:    "slice-5",
					mcc:          "001",
					mnc:          "01",
					sst:          "1",
					sd:           "050607",
					deviceGroups: []string{},
					upfHostname:  "upf.local",
				},
			},
			deviceGroups: nil,
			validateFunc: func(t *testing.T, sessions []nfConfigApi.SessionManagement) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				if len(sessions[0].GetIpDomain()) != 0 {
					t.Errorf("expected no IP domain due to empty device group list, got %+v", sessions[0].GetIpDomain())
				}
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
			if tt.name == "invalid upf hostname (non-string)" {
				slices[0].SiteInfo.Upf["upf-name"] = 1234
			}

			cfg := inMemoryConfig{}
			cfg.syncSessionManagement(slices, deviceGroupMap)

			if tt.validateFunc != nil {
				tt.validateFunc(t, cfg.sessionManagement)
			}
		})
	}
}
