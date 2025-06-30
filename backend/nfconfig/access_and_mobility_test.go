// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"fmt"
	"testing"

	"github.com/omec-project/webconsole/configmodels"
)

func generateNetworkSlice(mcc, mnc, sst, sd string, tacs []int32) configmodels.Slice {
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

// Two slices with same PLMN and SNSSAI but intersection TACs (one TAC is common)
// Expected: one AccessAndMobility object with set of TACs
func TestTwoSlicesSamePlmnSnssaiTacsIntersection(t *testing.T) {
	ns1 := generateNetworkSlice("001", "01", "01", "1", []int32{3, 4})
	ns2 := generateNetworkSlice("001", "01", "01", "1", []int32{4, 6})
	c := inMemoryConfig{}
	c.syncAccessAndMobility([]configmodels.Slice{ns1, ns2})
	if len(c.accessAndMobility) != 1 {
		t.Errorf("expected AccessAndMobility of length 1, got: %v", len(c.accessAndMobility))
	}
	if len(c.accessAndMobility[0].Tacs) != 3 {
		t.Errorf("expected Tacs of length 4, got: %v", len(c.accessAndMobility[0].Tacs))
	}
}

// Two slices with same PLMN and different SNSSAI
// Expected: two AccessAndMobility objects
func TestTwoSlicesSamePlmnDifferentSnssai(t *testing.T) {
	ns1 := generateNetworkSlice("001", "01", "01", "1", []int32{1})
	ns2 := generateNetworkSlice("001", "01", "02", "2", []int32{2})
	c := inMemoryConfig{}
	c.syncAccessAndMobility([]configmodels.Slice{ns1, ns2})
	if len(c.accessAndMobility) != 2 {
		t.Errorf("expected AccessAndMobility of length 2, got: %v", len(c.accessAndMobility))
	}
}

// Two slices with different PLMN and same SNSSAI
// Expected: two AccessAndMobility objects
func TestTwoSlicesDifferentPlmnSameSnssai(t *testing.T) {
	ns1 := generateNetworkSlice("001", "01", "01", "1", []int32{1})
	ns2 := generateNetworkSlice("002", "02", "01", "1", []int32{1})
	c := inMemoryConfig{}
	c.syncAccessAndMobility([]configmodels.Slice{ns1, ns2})
	if len(c.accessAndMobility) != 2 {
		t.Errorf("expected AccessAndMobility of length 2, got: %v", len(c.accessAndMobility))
	}
}
