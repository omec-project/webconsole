// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package webui_context

import (
	"testing"

	"github.com/omec-project/openapi/v2/models"
)

func TestGetNfOamUriUsesNfServiceList(t *testing.T) {
	profile := models.NFProfile{NfType: models.NFTYPE_AMF}
	profile.SetNfServiceList(map[string]models.NFService{
		"1": {
			ServiceName:     "namf-oam",
			NfServiceStatus: models.NFSERVICESTATUS_REGISTERED,
			ApiPrefix:       func() *string { s := "http://amf:8080"; return &s }(),
		},
	})
	if got := getNfOamUri(profile, "namf-oam"); got != "http://amf:8080" {
		t.Fatalf("expected URI from NfServiceList, got %q", got)
	}
}
