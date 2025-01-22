// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import "testing"

func TestValidateName(t *testing.T) {
	var testCases = []struct {
		name     string
		expected bool
	}{
		{"validName", true},
		{"Valid-Name", true},
		{"Valid_Name", true},
		{"{invalid_name}", false},
		{"invalid&name", false},
		{"invalidName(R)", false},
		{"", false},
	}

	for _, tc := range testCases {
		r := IsValidName(tc.name)
		if r != tc.expected {
			t.Errorf("%s", tc.name)
		}
	}
}

func TestValidateFQDN(t *testing.T) {
	var testCases = []struct {
		fqdn     string
		expected bool
	}{
		{"upf-external.sdcore.svc.cluster.local", true},
		{"my-upf.my-domain.com", true},
		{"www.my-upf.com", true},
		{"some-upf-name", false},
		{"1.2.3.4", false},
		{"{upf-external}.sdcore.svc.cluster.local", false},
		{"http://my-upf.my-domain.com", false},
		{"my-domain.com/my-upf", false},
		{"", false},
	}

	for _, tc := range testCases {
		r := IsValidFQDN(tc.fqdn)
		if r != tc.expected {
			t.Errorf("%s", tc.fqdn)
		}
	}
}
