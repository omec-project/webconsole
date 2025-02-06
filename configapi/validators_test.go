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
		r := isValidName(tc.name)
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
		r := isValidFQDN(tc.fqdn)
		if r != tc.expected {
			t.Errorf("%s", tc.fqdn)
		}
	}
}

func TestValidateUpfPort(t *testing.T) {
	var testCases = []struct {
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
	var testCases = []struct {
		tac      string
		expected bool
	}{
		{"123", true},
		{"7000", true},
		{"1", true},
		{"16777215", true},
		{"0", false},
		{"16777216", false},
		{"invalid", false},
		{"123ad", false},
		{"", false},
	}

	for _, tc := range testCases {
		r := isValidGnbTac(tc.tac)
		if r != tc.expected {
			t.Errorf("%s", tc.tac)
		}
	}
}
