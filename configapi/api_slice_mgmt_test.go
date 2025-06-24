// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"encoding/json"
	"fmt"
	"github.com/omec-project/webconsole/dbadapter"
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
        "tac": 1
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
    "sd": "1",
    "sst": "001"
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

func networkSliceWithGnbParams(gnbName string, gnbTac int32) string {
	gnb := configmodels.SliceSiteInfoGNodeBs{
		Name: gnbName,
		Tac:  gnbTac,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: "demo",
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{gnb},
	}
	slice := configmodels.Slice{
		SliceName: "slice-1",
		SiteInfo:  siteInfo,
	}
	sliceTmp, err := json.Marshal(slice)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal network slice: %v", err))
	}
	return string(sliceTmp[:])
}

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
			name:         "Device Group invalid name (invalid token)",
			route:        "/config/v1/device-group/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Device Group invalid name (invalid length)",
			route:        "/config/v1/device-group/" + genLongString(257),
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
			if tc.expectedCode == http.StatusOK {
				dbadapter.CommonDBClient = &MockMongoClientEmptyDB{}
			}
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
			name:         "Network Slice invalid name (invalid token)",
			route:        "/config/v1/network-slice/invalid&name",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Network Slice invalid name (invalid length)",
			route:        "/config/v1/network-slice/" + genLongString(257),
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
			if tc.expectedCode == http.StatusOK {
				dbadapter.CommonDBClient = &MockMongoClientEmptyDB{}
			}
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

func TestNetworkSlicePostHandler_NetworkSliceGnbTacValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	AddConfigV1Service(router)

	testCases := []struct {
		name         string
		route        string
		inputData    string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Network Slice invalid gNB name",
			route:        "/config/v1/network-slice/slice-1",
			inputData:    networkSliceWithGnbParams("", 3),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to create network slice slice-1. Please check the log for details."}`,
		},
		{
			name:         "Network Slice invalid gNB TAC",
			route:        "/config/v1/network-slice/slice-1",
			inputData:    networkSliceWithGnbParams("valid-gnb", 0),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to create network slice slice-1. Please check the log for details."}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origChannel := configChannel
			configChannel = make(chan *configmodels.ConfigMessage, 1)
			defer func() { configChannel = origChannel }()
			req, err := http.NewRequest(http.MethodPost, tc.route, strings.NewReader(tc.inputData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			if tc.expectedCode != w.Code {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedCode, w.Code)
			}
			if tc.expectedBody != w.Body.String() {
				t.Errorf("Expected `%v`, got `%v`", tc.expectedBody, w.Body.String())
			}
		})
	}
}
