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

func TestSyncSessionManagement_ValidSlice(t *testing.T) {
	// Slice with a valid SNSSAI, UPF and device group. The session management gets a valid config.
	slice := configmodels.Slice{
		SliceName: "slice-1",
		SliceId: configmodels.SliceSliceId{
			Sst: "1",
			Sd:  "010203",
		},
		SiteDeviceGroup: []string{"dg-1"},
		SiteInfo: configmodels.SliceSiteInfo{
			Plmn: configmodels.SliceSiteInfoPlmn{
				Mcc: "001",
				Mnc: "01",
			},
			Upf: map[string]interface{}{
				"upf-name": "upf.local",
			},
			GNodeBs: []configmodels.SliceSiteInfoGNodeBs{
				{Name: "gnb-1"},
			},
		},
	}

	deviceGroupMap := map[string]configmodels.DeviceGroups{
		"dg-1": {
			IpDomainExpanded: configmodels.DeviceGroupsIpDomainExpanded{
				Dnn:        "internet",
				DnsPrimary: "8.8.8.8",
				UeIpPool:   "10.1.1.0/24",
				Mtu:        1500,
			},
		},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement([]configmodels.Slice{slice}, deviceGroupMap)

	if len(cfg.sessionManagement) != 1 {
		t.Fatalf("expected 1 session config, got %d", len(cfg.sessionManagement))
	}

	session := cfg.sessionManagement[0]
	if got := session.GetSliceName(); got != "slice-1" {
		t.Errorf("expected slice name 'slice-1', got %s", got)
	}

	expectedPlmn := nfConfigApi.PlmnId{
		Mcc: "001",
		Mnc: "01",
	}

	if got := session.GetPlmnId(); got != expectedPlmn {
		t.Errorf("unexpected PLMN: got %+v, want %+v", got, expectedPlmn)
	}

	sd := "010203"
	expectedSnssai := nfConfigApi.Snssai{
		Sst: 1,
		Sd:  &sd,
	}
	if got := session.GetSnssai(); !reflect.DeepEqual(got, expectedSnssai) {
		t.Errorf("unexpected SNSSAI: got %+v, want %+v", got, expectedSnssai)
	}

	expectedUpf := nfConfigApi.Upf{Hostname: "upf.local"}
	if got := session.GetUpf(); got != expectedUpf {
		t.Errorf("unexpected UPF: got %+v, want %+v", got, expectedUpf)
	}

	expectedIpDomain := nfConfigApi.IpDomain{
		DnnName:  "internet",
		DnsIpv4:  "8.8.8.8",
		UeSubnet: "10.1.1.0/24",
		Mtu:      1500,
	}
	if got := session.GetIpDomain(); len(got) == 0 || got[0] != expectedIpDomain {
		t.Errorf("unexpected IP domain: got %+v, want %+v", got, []nfConfigApi.IpDomain{expectedIpDomain})
	}

	if got := session.GetGnbNames(); !reflect.DeepEqual(got, []string{"gnb-1"}) {
		t.Errorf("unexpected gNBs: got %+v, want %+v", got, []string{"gnb-1"})
	}
}

func TestSyncSessionManagement_InvalidSst(t *testing.T) {
	// Invalid SST, session management does not get a config.
	slice := configmodels.Slice{
		SliceName: "bad-slice",
		SliceId:   configmodels.SliceSliceId{Sst: "", Sd: "010203"},
		SiteInfo: configmodels.SliceSiteInfo{
			Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"},
			Upf:  map[string]interface{}{"upf-name": "upf.local"},
		},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement([]configmodels.Slice{slice}, nil)

	if len(cfg.sessionManagement) != 0 {
		t.Errorf("expected no session configs due to invalid SST, got %d", len(cfg.sessionManagement))
	}
}

func TestSyncSessionManagement_MissingUpf(t *testing.T) {
	// Missing UPF. The session management gets a config without UPF.
	slice := configmodels.Slice{
		SliceName: "no-upf",
		SliceId: configmodels.SliceSliceId{
			Sst: "1", Sd: "010203",
		},
		SiteDeviceGroup: []string{"dg-1"},
		SiteInfo: configmodels.SliceSiteInfo{
			Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"},
			// no UPF
			GNodeBs: []configmodels.SliceSiteInfoGNodeBs{{Name: "gnb-1"}},
		},
	}

	deviceGroupMap := map[string]configmodels.DeviceGroups{
		"dg-1": {
			IpDomainExpanded: configmodels.DeviceGroupsIpDomainExpanded{
				Dnn: "internet", DnsPrimary: "8.8.8.8", UeIpPool: "10.1.1.0/24", Mtu: 1500,
			},
		},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement([]configmodels.Slice{slice}, deviceGroupMap)

	if len(cfg.sessionManagement) != 1 {
		t.Fatalf("expected 1 session config, got %d", len(cfg.sessionManagement))
	}
	if cfg.sessionManagement[0].Upf.GetHostname() != "" {
		t.Errorf("expected UPF to be unset, got %+v", cfg.sessionManagement[0].GetUpf())
	}
}

func TestSyncSessionManagement_InvalidUpfHostname(t *testing.T) {
	// Invalid UPF hostname. The session management gets a config without UPF.
	slice := configmodels.Slice{
		SliceName:       "bad-upf",
		SliceId:         configmodels.SliceSliceId{Sst: "1", Sd: "010203"},
		SiteDeviceGroup: []string{"dg-1"},
		SiteInfo: configmodels.SliceSiteInfo{
			Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"},
			Upf:  map[string]interface{}{"upf-name": 123}, // Invalid type
		},
	}

	deviceGroupMap := map[string]configmodels.DeviceGroups{
		"dg-1": {
			IpDomainExpanded: configmodels.DeviceGroupsIpDomainExpanded{
				Dnn: "internet", DnsPrimary: "8.8.8.8", UeIpPool: "10.1.1.0/24", Mtu: 1500,
			},
		},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement([]configmodels.Slice{slice}, deviceGroupMap)

	if len(cfg.sessionManagement) != 1 {
		t.Fatalf("expected 1 session config, got %d", len(cfg.sessionManagement))
	}
	if cfg.sessionManagement[0].Upf.GetHostname() != "" {
		t.Errorf("expected UPF to be unset, got %+v", cfg.sessionManagement[0].GetUpf())
	}
}

func TestSyncSessionManagement_MissingDeviceGroup(t *testing.T) {
	// Missing device group. The session management gets a config without IPDomain.
	slice := configmodels.Slice{
		SliceName:       "no-dg",
		SliceId:         configmodels.SliceSliceId{Sst: "1", Sd: "010203"},
		SiteDeviceGroup: []string{"dg-x"}, // not in deviceGroupMap
		SiteInfo: configmodels.SliceSiteInfo{
			Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"},
			Upf:  map[string]interface{}{"upf-name": "upf.local"},
		},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement([]configmodels.Slice{slice}, map[string]configmodels.DeviceGroups{})

	if len(cfg.sessionManagement) != 1 {
		t.Fatalf("expected 1 session config, got %d", len(cfg.sessionManagement))
	}

	if len(cfg.sessionManagement[0].GetIpDomain()) != 0 {
		t.Errorf("expected no IP domains due to missing device group, got %+v", cfg.sessionManagement[0].GetIpDomain())
	}
}

func TestSyncSessionManagement_SortedBySliceName(t *testing.T) {
	// The session management gets a config sorted by slice name in ascending order.
	slices := []configmodels.Slice{
		{SliceName: "slice-b", SliceId: configmodels.SliceSliceId{Sst: "1", Sd: "010203"}, SiteInfo: configmodels.SliceSiteInfo{Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"}, Upf: map[string]interface{}{"upf-name": "upf.local"}}},
		{SliceName: "slice-a", SliceId: configmodels.SliceSliceId{Sst: "1", Sd: "010203"}, SiteInfo: configmodels.SliceSiteInfo{Plmn: configmodels.SliceSiteInfoPlmn{Mcc: "001", Mnc: "01"}, Upf: map[string]interface{}{"upf-name": "upf.local"}}},
	}

	cfg := &inMemoryConfig{}
	cfg.syncSessionManagement(slices, nil)

	if cfg.sessionManagement[0].GetSliceName() != "slice-a" {
		t.Errorf("expected slice-a first, got %s", cfg.sessionManagement[0].GetSliceName())
	}
}
