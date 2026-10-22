// SPDX-FileCopyrightText: 2026 Forsway Scandinavia AB
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfigFactoryChecksUpfRuleShapeLimit(t *testing.T) {
	origConfig := WebUIConfig
	defer func() { WebUIConfig = origConfig }()

	testCases := []struct {
		limit   int
		wantErr bool
	}{
		{-1, true},
		{0, false},
		{1, false},
		{MaxUpfRuleShapeLimit, false},
		{MaxUpfRuleShapeLimit + 1, true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("limit %d", tc.limit), func(t *testing.T) {
			WebUIConfig = &Config{Configuration: &Configuration{}}
			f := filepath.Join(t.TempDir(), "webuicfg.yaml")
			content := fmt.Sprintf("configuration:\n  mongodb:\n    url: mongodb://localhost:27017\n  upf-rule-shape-limit: %d\n", tc.limit)
			if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			err := InitConfigFactory(f)
			if (err != nil) != tc.wantErr {
				t.Errorf("limit %d: got error %v, want error %v", tc.limit, err, tc.wantErr)
			}
			if err == nil && WebUIConfig.Configuration.UpfRuleShapeLimit != tc.limit {
				t.Errorf("limit %d: read %d", tc.limit, WebUIConfig.Configuration.UpfRuleShapeLimit)
			}
		})
	}
}
