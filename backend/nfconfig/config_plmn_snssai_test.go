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

func makeNetworkSliceWithPlmnSnssai(mcc, mnc, sst string, sd string) configmodels.Slice {
	plmnId := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: "test",
		Plmn:     plmnId,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{},
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

func TestSyncPlmnSnssaiConfig_Success(t *testing.T) {
	tests := []struct {
		name               string
		slices             []configmodels.Slice
		expectedPlmnSnssai []nfConfigApi.PlmnSnssai
	}{
		{
			name: "Two slices same PLMN different S-NSSAI",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "23", "2", "abcd"),
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", "01234"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
		},
		{
			name: "Two slices same PLMN duplicate S-NSSAI",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", "01234"),
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", "01234"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Several slices different PLMN are ordered",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("999", "455", "2", "abcd"),
				makeNetworkSliceWithPlmnSnssai("123", "23", "3", "3333"),
				makeNetworkSliceWithPlmnSnssai("999", "455", "2", ""),
				makeNetworkSliceWithPlmnSnssai("123", "23", "3", "123"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(3, "123"),
						makeSnssaiWithSd(3, "3333"),
					},
				},
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "999", Mnc: "455"},
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(2),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
		},
		{
			name: "One slice no SD",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", ""),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(1),
					},
				},
			},
		},
		{
			name:               "Empty slices",
			slices:             []configmodels.Slice{},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := inMemoryConfig{}
			config.syncPlmnSnssai(tc.slices)
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, config.plmnSnssai) {
				t.Errorf("expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, config.plmnSnssai)
			}
		})
	}
}

func TestSyncPlmnSnssaiConfig_UnmarshalError_IgnoresNetworkSlice(t *testing.T) {
	tests := []struct {
		name               string
		slices             []configmodels.Slice
		expectedPlmnSnssai []nfConfigApi.PlmnSnssai
	}{
		{
			name: "Invalid SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", "01234"),
				makeNetworkSliceWithPlmnSnssai("123", "455", "a", "56789"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Empty SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "23", "1", "01234"),
				makeNetworkSliceWithPlmnSnssai("123", "455", "", "56789"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: nfConfigApi.PlmnId{Mcc: "123", Mnc: "23"},
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Invalid SST final list is empty",
			slices: []configmodels.Slice{
				makeNetworkSliceWithPlmnSnssai("123", "455", "a", "56789"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := inMemoryConfig{}
			config.syncPlmnSnssai(tc.slices)
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, config.plmnSnssai) {
				t.Errorf("expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, config.plmnSnssai)
			}
		})
	}
}
