// SPDX-License-Identifier: Apache-2.0

// +build ui

package ui

import (
	"embed"
)

//go:embed all:frontend_files
var FrontendFS embed.FS