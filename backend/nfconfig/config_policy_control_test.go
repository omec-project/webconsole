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

func makePolicyControlNetworkSlice(mcc, mnc, sst, sd string, dgs []string, filteringRules []configmodels.SliceApplicationFilteringRules) configmodels.Slice {
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
		SiteDeviceGroup:           dgs,
		SiteInfo:                  siteInfo,
		SliceId:                   sliceId,
		ApplicationFilteringRules: filteringRules,
	}
	return networkSlice
}

var (
	testSst             int32 = 1
	testSd                    = "12345"
	testRuleName              = "TestRule"
	testRulePriority    int32 = 12
	testRuleQci         int32 = 8
	testRuleArp         int32 = 100
	testMaxBrUl1              = "12 Kbps"
	testMaxBrDl1              = "67 Kbps"
	testMaxBrUl2              = "45 Kbps"
	testMaxBrDl2              = "12 Kbps"
	testDeviceGroupName       = "testDG"
	testDnnName               = "testDnn"
	testDG                    = configmodels.DeviceGroups{
		DeviceGroupName: testDeviceGroupName,
		Imsis:           []string{"001010123456789"},
		IpDomainExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			{
				Dnn: testDnnName,
			},
		},
	}
	testDG2 = configmodels.DeviceGroups{
		DeviceGroupName: "dg2",
		Imsis:           []string{"001010123456789"},
		IpDomainExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			{
				Dnn: "aDnn",
			},
		},
	}
	testDeviceGroups                   = map[string]configmodels.DeviceGroups{testDeviceGroupName: testDG}
	validSliceApplicationFilteringRule = configmodels.SliceApplicationFilteringRules{
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
	anotherSliceApplicationFilteringRule = configmodels.SliceApplicationFilteringRules{
		RuleName:       "SOME-RULE",
		Priority:       2,
		Action:         "deny",
		Endpoint:       "127.0.0.1",
		Protocol:       6,
		StartPort:      88,
		EndPort:        9000,
		AppMbrUplink:   45600,
		AppMbrDownlink: 12300,
		BitrateUnit:    "KBPS",
		TrafficClass: &configmodels.TrafficClassInfo{
			Qci: 9,
			Arp: 1,
		},
	}
)

func TestSyncPolicyControl(t *testing.T) {
	tests := []struct {
		name             string
		networkSlices    []configmodels.Slice
		deviceGroups     map[string]configmodels.DeviceGroups
		expectedResponse []nfConfigApi.PolicyControl
	}{
		{
			name: "Network Slice with valid SliceApplicationFilteringRules produces valid Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{validSliceApplicationFilteringRule}),
			},
			deviceGroups: testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId: *nfConfigApi.NewPlmnId("001", "01"),
					Snssai: makeSnssaiWithSd(testSst, testSd),
					Dnns:   []string{testDnnName},
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
								MaxBrUl: &testMaxBrUl1,
								MaxBrDl: &testMaxBrDl1,
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
			name: "Two network slices with valid SliceApplicationFilteringRules produces ordered valid Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("128", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG", "dg2"}, []configmodels.SliceApplicationFilteringRules{validSliceApplicationFilteringRule, anotherSliceApplicationFilteringRule}),
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: map[string]configmodels.DeviceGroups{"dg2": testDG2, testDeviceGroupName: testDG},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					Dnns:     []string{testDnnName},
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("128", "01"),
					Snssai: makeSnssaiWithSd(testSst, testSd),
					Dnns:   []string{"aDnn", testDnnName},
					PccRules: []nfConfigApi.PccRule{
						{
							RuleId: "SOME-RULE",
							Flows: []nfConfigApi.PccFlow{
								{
									Description: "permit out tcp from 127.0.0.1 to assigned 88-9000",
									Direction:   nfConfigApi.DIRECTION_BIDIRECTIONAL,
									Status:      nfConfigApi.STATUS_DISABLED,
								},
							},
							Qos: nfConfigApi.PccQos{
								FiveQi:  9,
								MaxBrUl: &testMaxBrUl2,
								MaxBrDl: &testMaxBrDl2,
								Arp: nfConfigApi.Arp{
									PriorityLevel: 1,
									PreemptCap:    nfConfigApi.PREEMPTCAP_MAY_PREEMPT,
									PreemptVuln:   nfConfigApi.PREEMPTVULN_PREEMPTABLE,
								},
							},
							Precedence: 2,
						},
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
								MaxBrUl: &testMaxBrUl1,
								MaxBrDl: &testMaxBrDl1,
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
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					Dnns:     []string{testDnnName},
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
			},
		},
		{
			name: "Network Slice without SliceApplicationFilteringRules produces default Policy Control config",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					Dnns:     []string{testDnnName},
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
			},
		},
		{
			name: "Network Slice with invalid SNSSAI is ignored",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("999", "99", "a", testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups:     testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{},
		},
		{
			name: "Network Slice with non-existent Device Group returns empty DNNs in Policy Control",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{"testDG"}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: map[string]configmodels.DeviceGroups{},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd),
					Dnns:     []string{},
					PccRules: []nfConfigApi.PccRule{*defaultPccRule},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := inMemoryConfig{}
			cfg.syncPolicyControl(tt.networkSlices, tt.deviceGroups)

			if !reflect.DeepEqual(cfg.policyControl, tt.expectedResponse) {
				t.Errorf("expected %+v, got %+v", tt.expectedResponse, cfg.policyControl)
			}
		})
	}
}
