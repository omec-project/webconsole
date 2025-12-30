// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd.

/*
 *  Metrics package is used to expose the metrics of the Webconsole service.
 */

package metrics

import (
	"net/http"

	"github.com/omec-project/webconsole/backend/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InitMetrics initializes Webconsole metrics
func InitMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.InitLog.Errorf("could not open metrics port: %v", err)
	}
}
