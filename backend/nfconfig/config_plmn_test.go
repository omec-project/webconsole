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

func makeNetworkSliceWithPlmn(mcc, mnc string) configmodels.Slice {
	plmnId := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: "test",
		Plmn:     plmnId,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{},
	}
	networkSlice := configmodels.Slice{
		SliceName: "slice1",
		SliceId:   configmodels.SliceSliceId{},
		SiteInfo:  siteInfo,
	}
	return networkSlice
}

func TestSyncPlmnConfig_Success(t *testing.T) {
	tests := []struct {
		name         string
		slices       []configmodels.Slice
		expectedPlmn []nfConfigApi.PlmnId
	}{
		{
			name: "Two slices different PLMN expects two elements",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmn("123", "23"),
				makeNetworkSliceWithPlmn("456", "77"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				{Mcc: "123", Mnc: "23"},
				{Mcc: "456", Mnc: "77"},
			},
		},
		{
			name: "Two slices different MNC expects two elements",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmn("123", "23"),
				makeNetworkSliceWithPlmn("123", "77"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				{Mcc: "123", Mnc: "23"},
				{Mcc: "123", Mnc: "77"},
			},
		},
		{
			name: "Two slices same PLMN expects one element",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmn("123", "23"),
				makeNetworkSliceWithPlmn("123", "23"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				{Mcc: "123", Mnc: "23"},
			},
		},
		{
			name: "Several slices different PLMN are ordered",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmn("999", "455"),
				makeNetworkSliceWithPlmn("123", "233"),
				makeNetworkSliceWithPlmn("999", "455"),
				makeNetworkSliceWithPlmn("123", "23"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				{Mcc: "123", Mnc: "23"},
				{Mcc: "123", Mnc: "233"},
				{Mcc: "999", Mnc: "455"},
			},
		},
		{
			name:         "Empty slices",
			slices:       []configmodels.Slice{},
			expectedPlmn: []nfConfigApi.PlmnId{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := inMemoryConfig{}
			config.syncPlmn(tc.slices)

			if !reflect.DeepEqual(tc.expectedPlmn, config.plmn) {
				t.Errorf("Expected PLMN %v, got %v", tc.expectedPlmn, config.plmn)
			}
		})
	}
}
