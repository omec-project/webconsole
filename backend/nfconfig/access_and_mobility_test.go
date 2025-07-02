// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"reflect"
	"testing"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

func TestAccessAndMobilityConfig(t *testing.T) {
	var c inMemoryConfig
	testCases := []struct {
		name                      string
		networkSlices             []configmodels.Slice
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name: "Two network slices with different PLMNs",
			networkSlices: []configmodels.Slice{
				makeNetworkSlice("002", "01", "001", "01", []int32{1, 2}),
				makeNetworkSlice("001", "01", "001", "02", []int32{3, 2}),
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
				makeNetworkSlice("001", "01", "001", "02", []int32{}),
				makeNetworkSlice("001", "01", "001", "01", []int32{}),
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
				makeNetworkSlice("001", "01", "001", "01", []int32{}),
				makeNetworkSlice("001", "01", "001", "", []int32{}),
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
				makeNetworkSlice("001", "01", "001", "01", []int32{1, 2}),
				makeNetworkSlice("001", "01", "001", "01", []int32{2, 3}),
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "001", Mnc: "01"},
					Snssai: makeSnssaiWithSd(1, "01"),
					Tacs:   []string{"1", "2", "3"},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalInMemoryConfig := c
			defer func() { c = originalInMemoryConfig }()
			c = inMemoryConfig{}
			c.syncAccessAndMobility(tc.networkSlices)
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, c.accessAndMobility) {
				t.Errorf("expected Access and Mobility: %#v, got: %#v", tc.expectedAccessAndMobility, c.accessAndMobility)
			}
		})
	}
}
