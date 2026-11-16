// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// backend/nfconfig decides whether to dump request bodies from GetLogger().Level(), so it has to
// follow SetLogLevel.
func TestSetLogLevel_IsVisibleThroughGetLogger(t *testing.T) {
	t.Cleanup(func() { component.SetLevel(zapcore.InfoLevel) })

	if got := GetLogger().Level(); got != zap.InfoLevel {
		t.Fatalf("initial level = %v, want info", got)
	}
	SetLogLevel(zapcore.DebugLevel)
	if got := GetLogger().Level(); got != zap.DebugLevel {
		t.Errorf("level after SetLogLevel(debug) = %v, want debug", got)
	}
	if !NfConfigLog.Desugar().Core().Enabled(zap.DebugLevel) {
		t.Error("category loggers did not follow SetLogLevel")
	}
}
