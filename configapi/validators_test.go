// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{genLongString(256), true},
		{"Valid-Name", true},
		{"Valid_Name", true},
		{"{invalid_name}", false},
		{"invalid&name", false},
		{"invalidName(R)", false},
		{"-invalidName", false},
		{"_invalidName", false},
		{"4invalidName", false},
		{"-_invalid", false},
		{"", false},
		{genLongString(257), false},
	}

	for _, tc := range testCases {
		r := isValidName(tc.name)
		if r != tc.expected {
			t.Errorf("%s", tc.name)
		}
	}
}

func TestValidateFQDN(t *testing.T) {
	testCases := []struct {
		fqdn     string
		expected bool
	}{
		{"upf-external.sdcore.svc.cluster.local", true},
		{"123-external.sdcore.svc.cluster.local", true},
		{"my-upf.my-domain.com", true},
		{"www.my-upf.com", true},
		{"some-upf-name", false},
		{"1.2.3.4", false},
		{"{upf-external}.sdcore.svc.cluster.local", false},
		{"http://my-upf.my-domain.com", false},
		{"my-domain.com/my-upf", false},
		{"-upf-external.sdcore.svc.cluster.local", false},
		{"", false},
	}

	for _, tc := range testCases {
		r := isValidFQDN(tc.fqdn)
		if r != tc.expected {
			t.Errorf("%s", tc.fqdn)
		}
	}
}

func TestValidateUpfPort(t *testing.T) {
	testCases := []struct {
		port     string
		expected bool
	}{
		{"123", true},
		{"7000", true},
		{"0", true},
		{"65535", true},
		{"-1", false},
		{"65536", false},
		{"invalid", false},
		{"123ad", false},
		{"", false},
	}

	for _, tc := range testCases {
		r := isValidUpfPort(tc.port)
		if r != tc.expected {
			t.Errorf("%s", tc.port)
		}
	}
}

func TestValidateGnbTac(t *testing.T) {
	testCases := []struct {
		tac      int32
		expected bool
	}{
		{123, true},
		{7000, true},
		{1, true},
		{16777215, true},
		{0, false},
		{16777216, false},
	}

	for _, tc := range testCases {
		r := isValidGnbTac(tc.tac)
		if r != tc.expected {
			t.Errorf("%d", tc.tac)
		}
	}
}

func genLongString(length int) string {
	return strings.Repeat("a", length)
}

// A PLMN with only one of MCC/MNC set is not a usable prefix: it is neither a genuinely
// unassigned PLMN (both empty) nor a complete one, so an IMSI must not be accepted against it.
func TestValidImsiForPlmn(t *testing.T) {
	testCases := []struct {
		name     string
		imsi     string
		mcc      string
		mnc      string
		expected bool
	}{
		{"matching PLMN prefix", "208930000000001", "208", "93", true},
		{"mismatching PLMN prefix", "111220000000001", "208", "93", false},
		{"both empty is unassigned and always accepted", "208930000000001", "", "", true},
		{"mcc set, mnc empty is rejected", "208930000000001", "208", "", false},
		{"mcc empty, mnc set is rejected", "208930000000001", "", "93", false},
	}

	for _, tc := range testCases {
		if r := isValidImsiForPlmn(tc.imsi, tc.mcc, tc.mnc); r != tc.expected {
			t.Errorf("%s: isValidImsiForPlmn(%q, %q, %q) = %v, want %v", tc.name, tc.imsi, tc.mcc, tc.mnc, r, tc.expected)
		}
	}
}

// A PLMN must be either fully unassigned or fully specified; one field set without the other
// cannot be completed later since a non-zero PLMN is treated as immutable once stored.
func TestIsCompletePlmn(t *testing.T) {
	testCases := []struct {
		name     string
		mcc      string
		mnc      string
		expected bool
	}{
		{"both set", "208", "93", true},
		{"both empty", "", "", true},
		{"mcc only", "208", "", false},
		{"mnc only", "", "93", false},
	}

	for _, tc := range testCases {
		if r := isCompletePlmn(tc.mcc, tc.mnc); r != tc.expected {
			t.Errorf("%s: isCompletePlmn(%q, %q) = %v, want %v", tc.name, tc.mcc, tc.mnc, r, tc.expected)
		}
	}
}

func TestValidateBitrate(t *testing.T) {
	testCases := []struct {
		name     string
		value    int32
		unit     string
		expected bool
	}{
		{"unset", 0, bitrateUnitMbps, true},
		{"within the field", 2000, bitrateUnitMbps, true},
		{"the largest rate that fits", 2147483, "kbps", true},
		{"negative", -1, bitrateUnitMbps, false},
		{"too large for the field", 3, bitrateUnitGbps, false},
		{"unset unit is read as bps", 2000000, "", true},
	}

	for _, tc := range testCases {
		if r := isValidBitrate(tc.value, tc.unit); r != tc.expected {
			t.Errorf("%s: isValidBitrate(%d, %q) = %v, want %v", tc.name, tc.value, tc.unit, r, tc.expected)
		}
	}
}

func TestValidateDeviceGroupBitrate(t *testing.T) {
	testCases := []struct {
		name     string
		value    int64
		unit     string
		expected bool
	}{
		{"a rate left unset", 0, bitrateUnitMbps, true},
		{"an ordinary rate", 100, bitrateUnitMbps, true},
		{"the largest rate that can be served", 65535, bitrateUnitGbps, true},
		{"one unit past it", 65536, bitrateUnitGbps, false},
		{"negative, which the old ingest path turned into the largest rate there is", -1, bitrateUnitMbps, false},
		// 10^10 Gbps is 10^19 bps, past what int64 holds: multiplying wraps it to a negative
		// number, so a bound compared against the product would accept it.
		{"a product that wraps the field", 10000000000, bitrateUnitGbps, false},
		{"unset unit is read as bps", 65535000000000, "", true},
	}

	for _, tc := range testCases {
		if r := isValidDeviceGroupBitrate(tc.value, tc.unit); r != tc.expected {
			t.Errorf("%s: isValidDeviceGroupBitrate(%d, %q) = %v, want %v", tc.name, tc.value, tc.unit, r, tc.expected)
		}
	}
}
