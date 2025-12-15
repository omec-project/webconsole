// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"strconv"
	"testing"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Test_filterExistingIMSIsFromAuthDB(t *testing.T) {
	origAuth := dbadapter.AuthDBClient
	defer func() { dbadapter.AuthDBClient = origAuth }()

	dbadapter.AuthDBClient = &dbadapter.MockDBClient{
		GetManyFn: func(collName string, filter bson.M) ([]map[string]any, error) {
			if collName != AuthSubsDataColl {
				t.Fatalf("expected coll %s, got %s", AuthSubsDataColl, collName)
			}
			// Return only one existing subscriber
			return []map[string]any{{"ueId": "imsi-002"}}, nil
		},
	}

	got, err := filterExistingIMSIsFromAuthDB([]string{"001", "002", "003"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "002" {
		t.Fatalf("expected [002], got %#v", got)
	}
}

func Test_updatePolicyAndProvisionedDataBatch_UsesPutMany(t *testing.T) {
	origCommon := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommon }()

	putManyCalls := make([]string, 0)
	dbadapter.CommonDBClient = &dbadapter.MockDBClient{
		PutManyFn: func(collName string, filterArray []primitive.M, putDataArray []map[string]any) error {
			putManyCalls = append(putManyCalls, collName)
			if len(filterArray) != 2 || len(putDataArray) != 2 {
				t.Fatalf("expected 2 items, got filters=%d docs=%d", len(filterArray), len(putDataArray))
			}
			// basic sanity: ueId is present
			if putDataArray[0]["ueId"] == nil || putDataArray[1]["ueId"] == nil {
				t.Fatalf("expected ueId in docs, got %#v", putDataArray)
			}
			return nil
		},
	}

	snssai := &models.Snssai{Sst: 1, Sd: "010203"}
	qos := &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{DnnMbrDownlink: 1000, DnnMbrUplink: 1000}

	err := updatePolicyAndProvisionedDataBatch([]string{"001", "002"}, "208", "93", snssai, "internet", qos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect one bulk call per collection touched.
	if len(putManyCalls) != 5 {
		t.Fatalf("expected 5 PutMany calls, got %d (%v)", len(putManyCalls), putManyCalls)
	}
}

func Test_updatePolicyAndProvisionedDataBatch_ChunksBy1000(t *testing.T) {
	origCommon := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = origCommon }()

	callSizes := make([]int, 0)
	dbadapter.CommonDBClient = &dbadapter.MockDBClient{
		PutManyFn: func(collName string, filterArray []primitive.M, putDataArray []map[string]any) error {
			if len(filterArray) != len(putDataArray) {
				t.Fatalf("filters/docs mismatch: filters=%d docs=%d", len(filterArray), len(putDataArray))
			}
			if len(filterArray) > 1000 {
				t.Fatalf("expected chunk size <= 1000, got %d", len(filterArray))
			}
			callSizes = append(callSizes, len(filterArray))
			return nil
		},
	}

	imsis := make([]string, 0, 1001)
	for i := 0; i < 1001; i++ {
		imsis = append(imsis, strconv.Itoa(i))
	}

	snssai := &models.Snssai{Sst: 1, Sd: "010203"}
	qos := &configmodels.DeviceGroupsIpDomainExpandedUeDnnQos{DnnMbrDownlink: 1000, DnnMbrUplink: 1000}

	err := updatePolicyAndProvisionedDataBatch(imsis, "208", "93", snssai, "internet", qos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1001 IMSIs => 2 chunks, and we call PutMany 5 times per chunk.
	if len(callSizes) != 10 {
		t.Fatalf("expected 10 PutMany calls (2 chunks x 5 collections), got %d", len(callSizes))
	}
	// Expect five 1000-sized calls and five 1-sized calls (order grouped by chunk).
	count1000 := 0
	count1 := 0
	for _, s := range callSizes {
		switch s {
		case 1000:
			count1000++
		case 1:
			count1++
		default:
			t.Fatalf("unexpected call size %d", s)
		}
	}
	if count1000 != 5 || count1 != 5 {
		t.Fatalf("expected five 1000-sized and five 1-sized calls; got 1000=%d 1=%d", count1000, count1)
	}
}
