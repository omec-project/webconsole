// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

package nfconfig

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/webconsole/configmodels"
)

const (
	testSst                int32 = 1
	testSd                       = "12345"
	testSd2                      = "010203"
	testRuleName                 = "TestRule"
	testRulePriority       int32 = 12
	testRuleQci            int32 = 8
	testRuleArp            int32 = 100
	testDeviceGroupName          = "testDG"
	testDeviceGroupNameDG2       = "dg2"
	testDnnName                  = "testDnn"
	// The rates these describe -- 12345, 67890, 45600 and 12300 bps -- are not whole numbers of
	// Kbps, so they are truncated to the Kbps below. Serving the exact bps instead would be read
	// as Mbps by the SMF, which is why the truncation stays where no larger unit is exact.
	testMaxBrUl1 = "12 Kbps"
	testMaxBrDl1 = "67 Kbps"
	testMaxBrUl2 = "45 Kbps"
	testMaxBrDl2 = "12 Kbps"
)

func makePolicyControlNetworkSlice(mcc, mnc, sst, sd string, dgs []string, filteringRules []configmodels.SliceApplicationFilteringRules) configmodels.Slice {
	plmnId := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: testSiteName,
		Plmn:     plmnId,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{},
	}
	sliceId := configmodels.SliceSliceId{
		Sst: sst,
		Sd:  sd,
	}
	networkSlice := configmodels.Slice{
		SliceName:                 testSliceName1,
		SiteDeviceGroup:           dgs,
		SiteInfo:                  siteInfo,
		SliceId:                   sliceId,
		ApplicationFilteringRules: filteringRules,
	}
	return networkSlice
}

var (
	testDG = configmodels.DeviceGroups{
		DeviceGroupName: testDeviceGroupName,
		Imsis:           []string{imsiTest},
		IpDomainsExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
			{
				Dnn: testDnnName,
			},
		},
	}
	testDG2 = configmodels.DeviceGroups{
		DeviceGroupName: testDeviceGroupNameDG2,
		Imsis:           []string{imsiTest},
		IpDomainsExpanded: []configmodels.DeviceGroupsIpDomainExpanded{
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
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd2, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{validSliceApplicationFilteringRule}),
			},
			deviceGroups: testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId: *nfConfigApi.NewPlmnId("001", "01"),
					Snssai: makeSnssaiWithSd(testSst, testSd2),
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
								MaxBrUl: nfConfigApi.PtrString(testMaxBrUl1),
								MaxBrDl: nfConfigApi.PtrString(testMaxBrDl1),
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
				makePolicyControlNetworkSlice("128", "01", fmt.Sprintf("%d", testSst), testSd, []string{testDeviceGroupName, testDeviceGroupNameDG2}, []configmodels.SliceApplicationFilteringRules{validSliceApplicationFilteringRule, anotherSliceApplicationFilteringRule}),
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: map[string]configmodels.DeviceGroups{testDeviceGroupNameDG2: testDG2, testDeviceGroupName: testDG},
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
								MaxBrUl: nfConfigApi.PtrString(testMaxBrUl2),
								MaxBrDl: nfConfigApi.PtrString(testMaxBrDl2),
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
								MaxBrUl: nfConfigApi.PtrString(testMaxBrUl1),
								MaxBrDl: nfConfigApi.PtrString(testMaxBrDl1),
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
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{}),
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
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{}),
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
				makePolicyControlNetworkSlice("999", "99", "a", testSd2, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups:     testDeviceGroups,
			expectedResponse: []nfConfigApi.PolicyControl{},
		},
		{
			name: "Network Slice with non-existent Device Group returns empty DNNs in Policy Control",
			networkSlices: []configmodels.Slice{
				makePolicyControlNetworkSlice("001", "01", fmt.Sprintf("%d", testSst), testSd2, []string{testDeviceGroupName}, []configmodels.SliceApplicationFilteringRules{}),
			},
			deviceGroups: map[string]configmodels.DeviceGroups{},
			expectedResponse: []nfConfigApi.PolicyControl{
				{
					PlmnId:   *nfConfigApi.NewPlmnId("001", "01"),
					Snssai:   makeSnssaiWithSd(testSst, testSd2),
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

func ruleWithGuaranteedRates(gbrUl, gbrDl int32) configmodels.SliceApplicationFilteringRules {
	rule := validSliceApplicationFilteringRule
	rule.AppGbrUplink = gbrUl
	rule.AppGbrDownlink = gbrDl
	return rule
}

// Rates reach buildPccQos already normalised to bps, and ConvertToString renders them from there —
// so 10 here really is 10 bps, which is below a Kbps and is rendered as such.
func TestBuildPccQosCarriesGuaranteedBitRate(t *testing.T) {
	qos := buildPccQos(ruleWithGuaranteedRates(10, 20))

	if !qos.HasGbrUl() || !qos.HasGbrDl() {
		t.Fatal("a configured guaranteed rate must reach the policy served to the PCF")
	}
	if got := qos.GetGbrUl(); got != "10 bps" {
		t.Errorf("gbrUl = %q, want %q", got, "10 bps")
	}
	if got := qos.GetGbrDl(); got != "20 bps" {
		t.Errorf("gbrDl = %q, want %q", got, "20 bps")
	}
	if got := qos.GetMaxBrUl(); got != testMaxBrUl1 {
		t.Errorf("maxBrUl = %q, want the maximum rates unaffected", got)
	}
}

// A rule with no guaranteed rate must not acquire one. Non-GBR flows are the common case and a
// zero must not be served as a guarantee of zero.
func TestBuildPccQosOmitsAnUnsetGuaranteedBitRate(t *testing.T) {
	qos := buildPccQos(ruleWithGuaranteedRates(0, 0))

	if qos.HasGbrUl() || qos.HasGbrDl() {
		t.Error("a rule with no guaranteed rate must not report one")
	}
}

// A guarantee in one direction only is plausible where the return link is the scarce one, so it
// must survive rather than being dropped for being incomplete.
func TestBuildPccQosCarriesAOneDirectionalGuarantee(t *testing.T) {
	qos := buildPccQos(ruleWithGuaranteedRates(10, 0))

	if !qos.HasGbrUl() {
		t.Error("an uplink guarantee must be carried")
	}
	if qos.HasGbrDl() {
		t.Error("an unset downlink guarantee must stay unset")
	}
}
