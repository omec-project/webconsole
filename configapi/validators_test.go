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
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteString("a")
	}
	return sb.String()
}
