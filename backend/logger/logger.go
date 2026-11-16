// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	utilLogger "github.com/omec-project/util/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var component = utilLogger.NewComponent("WebUI")

var (
	AppLog      = component.Category("App")
	InitLog     = component.Category("Init")
	WebUILog    = component.Category("WebUI")
	ContextLog  = component.Category("Context")
	GinLog      = component.Category("GIN")
	ConfigLog   = component.Category("CONFIG")
	DbLog       = component.Category("DB")
	AuthLog     = component.Category("Auth")
	NfConfigLog = component.Category("NFConfig")
)

func GetLogger() *zap.Logger {
	return component.Zap()
}

// SetLogLevel: set the log level (panic|fatal|error|warn|info|debug)
func SetLogLevel(level zapcore.Level) {
	InitLog.Infoln("set log level:", level)
	component.SetLevel(level)
}
