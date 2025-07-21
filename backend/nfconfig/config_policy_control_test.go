// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package nfconfig

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

func makePolicyControlNetworkSlice(mcc, mnc, sst, sd string, filteringRules []configmodels.SliceApplicationFilteringRules) configmodels.Slice {
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
		SliceName:                 "slice1",
		SiteInfo:                  siteInfo,
		SliceId:                   sliceId,
		ApplicationFilteringRules: filteringRules,
	}
	return networkSlice
}

var (
	testSst                            int32 = 1
	testSd                                   = "12345"
	testRuleName                             = "TestRule"
	testRulePriority                   int32 = 12
	testRuleQci                        int32 = 12
	testRuleArp                        int32 = 100
	validSliceApplicationFilteringRule       = configmodels.SliceApplicationFilteringRules{
		RuleName:       testRuleName,
		Priority:       testRulePriority,
		Action:         "allow",
		Endpoint:       "0.0.0.0",
		Protocol:       17,
		StartPort:      5,
		EndPort:        5555,
		AppMbrUplink:   12345,
		AppMbrDownlink: 67890,
		BitrateUnit:    "KBPS",
		TrafficClass: &configmodels.TrafficClassInfo{
			Qci: testRuleQci,
			Arp: testRuleArp,
		},
	}
)

func TestSyncPolicyControl(t *testing.T) {
	tests := []struct {
		name             string
		networkSlices    []configmodels.Slice
		expectedResponse []nfConfigApi.PolicyControl
	}{
		{
			name: "Network Slice with valid SliceApplicationFilteringRules produces valid Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []configmodels.SliceApplicationFilteringRules{validSliceApplicationFilteringRule}),
			},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId: *nfConfigApi.NewPlmnId("001", "01"),
					Snssai: makeSnssaiWithSd(testSst, testSd),
					PccRules: []nfConfigApi.PccRule{
						{
							RuleId: testRuleName,
							Flows: []nfConfigApi.PccFlow{
								{
									Description: "permit out udp from any to assigned 5-5555",
									Direction:   nfConfigApi.DIRECTION_BIDIRECTIONAL,
									Status:      nfConfigApi.STATUS_ENABLED,
								},
							},
							Qos: nfConfigApi.PccQos{
								FiveQi:  testRuleQci,
								MaxBrUl: "12 Kbps",
								MaxBrDl: "67 Kbps",
								Arp: nfConfigApi.Arp{
									PriorityLevel: testRuleArp,
									PreemptCap:    nfConfigApi.PREEMPTCAP_MAY_PREEMPT,
									PreemptVuln:   nfConfigApi.PREEMPTVULN_PREEMPTABLE,
								},
							},
							Precedence: testRulePriority,
						},
					},
				},
			},
		},
		{
			name: "Network Slice without SliceApplicationFilteringRules produces default Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []configmodels.SliceApplicationFilteringRules{}),
			},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
			},
		},
		{
			name: "Network Slice without SliceApplicationFilteringRules produces default Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []configmodels.SliceApplicationFilteringRules{}),
			},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
			},
		},
		{
			name: "Network Slice with invalid SNSSAI is ignored",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("999", "99", "a", testSd, []configmodels.SliceApplicationFilteringRules{}),
			},
			expectedResponse: []nfConfigApi.PolicyControl{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := inMemoryConfig{}
			cfg.syncPolicyControl(tt.networkSlices)

			if !reflect.DeepEqual(cfg.policyControl, tt.expectedResponse) {
				t.Errorf("expected %+v, got %+v", tt.expectedResponse, cfg.policyControl)
			}
		})
	}
}
