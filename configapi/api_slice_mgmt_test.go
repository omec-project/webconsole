// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configmodels"
)

const NETWORK_SLICE_CONFIG = `{
  "application-filtering-rules": [
    {
      "action": "string",
      "app-mbr-downlink": 0,
      "app-mbr-uplink": 0,
      "bitrate-unit": "string",
      "dest-port-end": 0,
      "dest-port-start": 0,
      "endpoint": "string",
      "priority": 0,
      "protocol": 0,
      "rule-name": "string",
      "rule-trigger": "string",
      "traffic-class": {
        "arp": 0,
        "name": "string",
        "pdb": 0,
        "pelr": 0,
        "qci": 0
      }
    }
  ],
  "site-device-group": [
    "string"
  ],
  "site-info": {
    "gNodeBs": [
      {
        "name": "string",
        "tac": 0
      }
    ],
    "plmn": {
      "mcc": "string",
      "mnc": "string"
    },
    "site-name": "string",
    "upf": {
      "additionalProp1": {}
    }
  },
  "slice-id": {
    "sd": "string",
    "sst": "string"
  },
  "sliceName": "string"
}`
const DEVICE_GROUP_CONFIG = `{
  "group-name": "string",
  "imsis": [
    "string"
  ],
  "ip-domain-expanded": {
    "dnn": "string",
    "dns-primary": "string",
    "dns-secondary": "string",
    "mtu": 0,
    "ue-dnn-qos": {
      "bitrate-unit": "string",
      "dnn-mbr-downlink": 0,
      "dnn-mbr-uplink": 0,
      "traffic-class": {
        "arp": 0,
        "name": "string",
        "pdb": 0,
        "pelr": 0,
        "qci": 0
      }
    },
    "ue-ip-pool": "string"
  },
  "ip-domain-name": "string",
  "site-info": "string"
}`

func TestDeviceGroupPostHandler_DeviceGroupNameValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		expectedCode int
	}{
		{
			name:         "Device Group invalid name",
			route:        "/config/v1/device-group/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Device Group valid name",
			route:        "/config/v1/device-group/valid-devicegroup",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 1)
			defer func() { configChannel = origChannel }()
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(DEVICE_GROUP_CONFIG))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
		})
	}
}

func TestNetworkSlicePostHandler_NetworkSliceNameValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		expectedCode int
	}{
		{
			name:         "Network Slice invalid name",
			route:        "/config/v1/network-slice/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Network Slice valid name",
			route:        "/config/v1/network-slice/slice1",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 1)
			defer func() { configChannel = origChannel }()
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(NETWORK_SLICE_CONFIG))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
		})
	}
}
