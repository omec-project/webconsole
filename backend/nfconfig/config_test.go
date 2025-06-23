// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type MockDBClient struct {
	dbadapter.DBInterface
	Slices []configmodels.Slice
	err    error
}

func (m *MockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	for _, s := range m.Slices {
		ns := configmodels.ToBsonM(s)
		if ns == nil {
			panic("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, m.err
}

func makeNetworkSlice(mcc, mnc, sst, sd string) configmodels.Slice {
	upf := make(map[string]interface{}, 0)
	upf["upf-name"] = "upf"
	upf["upf-port"] = "8805"
	plmn := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	gnodeb := configmodels.SliceSiteInfoGNodeBs{
		Name: "demo-gnb1",
		Tac:  1,
	}
	slice_id := configmodels.SliceSliceId{
		Sst: sst,
		Sd:  sd,
	}
	site_info := configmodels.SliceSiteInfo{
		SiteName: "demo",
		Plmn:     plmn,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{gnodeb},
		Upf:      upf,
	}
	slice := configmodels.Slice{
		SliceName:       "slice1",
		SliceId:         slice_id,
		SiteDeviceGroup: []string{"group1", "group2"},
		SiteInfo:        site_info,
	}
	return slice
}

func makeSnssaiWithSd(sst int32, sd string) nfConfigApi.Snssai {
	s := nfConfigApi.NewSnssai(sst)
	s.SetSd(sd)
	return *s
}

func TestTriggerSync_Success(t *testing.T) {
	n := &NFConfigServer{}

	called := false
	originalSyncInMemoryFunc := syncInMemoryConfigFunc
	defer func() { syncInMemoryConfigFunc = originalSyncInMemoryFunc }()
	syncInMemoryConfigFunc = func(n *NFConfigServer) error {
		called = true
		return nil
	}

	n.TriggerSync()
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Fatal("expected syncInMemoryConfig to be called")
	}
}

func TestTriggerSync_RetryAndThenSuccess(t *testing.T) {
	n := &NFConfigServer{}

	callCount := 0
	originalSyncInMemoryFunc := syncInMemoryConfigFunc
	defer func() { syncInMemoryConfigFunc = originalSyncInMemoryFunc }()
	syncInMemoryConfigFunc = func(n *NFConfigServer) error {
		callCount++
		if callCount < 3 {
			return fmt.Errorf("mock error")
		}
		return nil
	}

	n.TriggerSync()

	time.Sleep(10 * time.Second)
	if callCount != 3 {
		t.Fatalf("expected 3 calls to syncInMemoryConfigFunc, got %d", callCount)
	}
}

func TestSyncPlmnSnssaiConfig_Success(t *testing.T) {
	tests := []struct {
		name               string
		slices             []configmodels.Slice
		expectedPlmn       []nfConfigApi.PlmnId
		expectedPlmnSnssai []nfConfigApi.PlmnSnssai
	}{
		{
			name: "Two slices same PLMN different S-NSSAI",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234"),
				makeNetworkSlice("123", "23", "2", "abcd"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
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
				makeNetworkSlice("123", "23", "1", "01234"),
				makeNetworkSlice("123", "23", "1", "01234"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Sereral slices different PLMN are ordered",
			slices: []configmodels.Slice{
				makeNetworkSlice("999", "455", "2", "abcd"),
				makeNetworkSlice("123", "23", "3", "3333"),
				makeNetworkSlice("999", "455", "2", ""),
				makeNetworkSlice("123", "23", "3", "123"),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
				*nfConfigApi.NewPlmnId("999", "455"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(3, "123"),
						makeSnssaiWithSd(3, "3333"),
					},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("999", "455"),
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(2),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
		},
		{
			name:               "Empty slices",
			slices:             []configmodels.Slice{},
			expectedPlmn:       []nfConfigApi.PlmnId{},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{},
		},
		{
			name: "One slice no SD",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", ""),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(1),
					},
				},
			},
		},
		{
			name:               "Empty slices",
			slices:             []configmodels.Slice{},
			expectedPlmn:       []nfConfigApi.PlmnId{},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: tc.slices,
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{},
			}

			err := n.syncInMemoryConfig()
			if err != nil {
				t.Errorf("expected no error. Got %s", err)
			}
			if !reflect.DeepEqual(tc.expectedPlmn, n.inMemoryConfig.plmn) {
				t.Errorf("Expected PLMN %v, got %v", tc.expectedPlmn, n.inMemoryConfig.plmn)
			}
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai)
			}
		})
	}
}

func TestSyncPlmnSnssaiConfig_DBError_KeepsPreviousConfig(t *testing.T) {
	tests := []struct {
		name               string
		expectedPlmn       []nfConfigApi.PlmnId
		expectedPlmnSnssai []nfConfigApi.PlmnSnssai
	}{
		{
			name:               "Initial empty PLMN S-NSSAI config",
			expectedPlmn:       []nfConfigApi.PlmnId{},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{},
		},
		{
			name: "Initial not empty PLMN S-NSSAI config",
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("44", "22"),
				*nfConfigApi.NewPlmnId("167", "24"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{makeNetworkSlice("999", "99", "9", "999")},
				err:    fmt.Errorf("mock error"),
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{
					plmn:       tc.expectedPlmn,
					plmnSnssai: tc.expectedPlmnSnssai,
				},
			}

			err := n.syncInMemoryConfig()

			if err == nil {
				t.Errorf("expected error. Got nil")
			}
			if !reflect.DeepEqual(tc.expectedPlmn, n.inMemoryConfig.plmn) {
				t.Errorf("Expected PLMN %v, got %v", tc.expectedPlmn, n.inMemoryConfig.plmn)
			}
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai)
			}
		})
	}
}

func TestSyncPlmnSnssaiConfig_UnmarshalError_IgnoresNetworkSlice(t *testing.T) {
	tests := []struct {
		name           string
		slices         []configmodels.Slice
		expectedResult []nfConfigApi.PlmnSnssai
	}{
		{
			name: "Invalid SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234"),
				makeNetworkSlice("123", "455", "a", "56789"),
			},
			expectedResult: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Empty SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234"),
				makeNetworkSlice("123", "455", "", "56789"),
			},
			expectedResult: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
		},
		{
			name: "Invalid SST final list is empty",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "455", "a", "56789"),
			},
			expectedResult: []nfConfigApi.PlmnSnssai{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: tc.slices,
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{},
			}

			err := n.syncInMemoryConfig()
			if err != nil {
				t.Errorf("expected no error. Got %s", err)
			}
			if !reflect.DeepEqual(tc.expectedResult, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedResult, n.inMemoryConfig.plmnSnssai)
			}
		})
	}
}
