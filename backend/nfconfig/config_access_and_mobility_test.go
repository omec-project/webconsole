// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

func makeAccessAndMobilityNetworkSlice(mcc, mnc, sst string, sd string, tacs []int32) configmodels.Slice {
	plmnId := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: "test",
		Plmn:     plmnId,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{},
	}
	for _, tac := range tacs {
		gNodeB := configmodels.SliceSiteInfoGNodeBs{
			Name: fmt.Sprintf("test-gnb-%d", tac),
			Tac:  tac,
		}
		siteInfo.GNodeBs = append(siteInfo.GNodeBs, gNodeB)
	}
	sliceId := configmodels.SliceSliceId{
		Sst: sst,
		Sd:  sd,
	}
	networkSlice := configmodels.Slice{
		SliceName: "slice1",
		SliceId:   sliceId,
		SiteInfo:  siteInfo,
	}
	return networkSlice
}

func TestAccessAndMobilityConfig(t *testing.T) {
	testCases := []struct {
		name                      string
		networkSlices             []configmodels.Slice
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name: "Two network slices with different PLMNs",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("002", "01", "001", "01", []int32{1, 2}),
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "02", []int32{3, 2}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "02"),
					Tacs:   []string{"2", "3"},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "002", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "01"),
					Tacs:   []string{"1", "2"},
				},
			},
		},
		{
			name: "Two network slices with same PLMN and different SNSSAI (SST and SD populated)",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "02", []int32{}),
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "01", []int32{}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "01"),
					Tacs:   []string{},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "02"),
					Tacs:   []string{},
				},
			},
		},
		{
			name: "Two network slices with same PLMN and different SNSSAI (only SST populated)",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "01", []int32{}),
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "", []int32{}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: nfConfigApi.Snssai{Sst: 1},
					Tacs:   []string{},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "01"),
					Tacs:   []string{},
				},
			},
		},
		{
			name: "Two network slices with same PLMN and same SNSSAI",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "01", []int32{1, 2}),
				makeAccessAndMobilityNetworkSlice("001", "01", "001", "01", []int32{2, 3}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "01"),
					Tacs:   []string{"1", "2", "3"},
				},
			},
		},
		{
			name: "Several slices different PLMN are ordered",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("999", "455", "2", "abcd", []int32{2, 1}),
				makeAccessAndMobilityNetworkSlice("123", "23", "3", "3333", []int32{4, 5, 1}),
				makeAccessAndMobilityNetworkSlice("999", "455", "2", "", []int32{1}),
				makeAccessAndMobilityNetworkSlice("123", "23", "3", "123", []int32{1}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					Snssai: makeSnssaiWithSd(3, "123"),
					Tacs:   []string{"1"},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					Snssai: makeSnssaiWithSd(3, "3333"),
					Tacs:   []string{"1", "4", "5"},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "999", Mnc: "455"},
					Snssai: *nfConfigApi.NewSnssai(2),
					Tacs:   []string{"1"},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "999", Mnc: "455"},
					Snssai: makeSnssaiWithSd(2, "abcd"),
					Tacs:   []string{"1", "2"},
				},
			},
		},
		{
			name:                      "Empty slices",
			networkSlices:             []configmodels.Slice{},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := inMemoryConfig{}
			c.syncAccessAndMobility(tc.networkSlices)
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, c.accessAndMobility) {
				t.Errorf("expected Access and Mobility: %#v, got: %#v", tc.expectedAccessAndMobility, c.accessAndMobility)
			}
		})
	}
}

func TestAccessAndMobilityConfig_UnmarshalError_IgnoresNetworkSlice(t *testing.T) {
	tests := []struct {
		name                      string
		networkSlices             []configmodels.Slice
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name: "Invalid SST is ignored",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("123", "23", "1", "01234", []int32{1}),
				makeAccessAndMobilityNetworkSlice("123", "455", "a", "56789", []int32{1}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "Empty SST is ignored",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("123", "23", "1", "01234", []int32{1}),
				makeAccessAndMobilityNetworkSlice("123", "455", "", "56789", []int32{1}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "Invalid SST final list is empty",
			networkSlices: []configmodels.Slice{
				makeAccessAndMobilityNetworkSlice("123", "455", "a", "56789", []int32{1}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := inMemoryConfig{}
			c.syncAccessAndMobility(tc.networkSlices)
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, c.accessAndMobility) {
				t.Errorf("expected Access and Mobility %v, got %v", tc.expectedAccessAndMobility, c.accessAndMobility)
			}
		})
	}
}
