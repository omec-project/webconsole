// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/nfConfigApi"
)

func TestGetImsiQosConfig(t *testing.T) {
	tests := []struct {
		name         string
		imsi         string
		inMemoryData []imsiQosConfig
		expectedCode int
		expectedData []nfConfigApi.ImsiQos
	}{
		{
			name: "matching dnn and imsi found",
			imsi: "imsi-001010000000001",
			inMemoryData: []imsiQosConfig{
				{
					dnn:   "internet",
					imsis: []string{"001010000000001"},
					qos: []nfConfigApi.ImsiQos{
						{MbrUplink: "20 Kbps",
							MbrDownlink:      "100 Kbps",
							FiveQi:           7,
							ArpPriorityLevel: 32,
						},
					},
				},
			},
			expectedCode: http.StatusOK,
			expectedData: []nfConfigApi.ImsiQos{
				{MbrUplink: "20 Kbps",
					MbrDownlink:      "100 Kbps",
					FiveQi:           7,
					ArpPriorityLevel: 32,
				},
			},
		},
		{
			name: "matching dnn but no imsi found",
			imsi: "imsi-001010000000001",
			inMemoryData: []imsiQosConfig{
				{
					dnn:   "internet",
					imsis: []string{"999990000000000"},
					qos: []nfConfigApi.ImsiQos{
						{MbrUplink: "20 Kbps",
							MbrDownlink:      "100 Kbps",
							FiveQi:           7,
							ArpPriorityLevel: 32,
						},
					},
				},
			},
			expectedCode: http.StatusNotFound,
			expectedData: []nfConfigApi.ImsiQos{},
		},
		{
			name: "matching imsi but dnn not found",
			imsi: "imsi-001010000000001",
			inMemoryData: []imsiQosConfig{
				{
					dnn:   "internet2",
					imsis: []string{"001010000000001"},
					qos: []nfConfigApi.ImsiQos{
						{MbrUplink: "20 Kbps",
							MbrDownlink:      "100 Kbps",
							FiveQi:           7,
							ArpPriorityLevel: 32,
						},
					},
				},
			},
			expectedCode: http.StatusNotFound,
			expectedData: []nfConfigApi.ImsiQos{},
		},
		{
			name:         "empty in memory config",
			imsi:         "imsi-999990000000000",
			inMemoryData: nil,
			expectedCode: http.StatusNotFound,
			expectedData: []nfConfigApi.ImsiQos{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			router := gin.New()

			nfServer := &NFConfigServer{
				Router:         router,
				inMemoryConfig: inMemoryConfig{imsiQos: tc.inMemoryData},
			}
			nfServer.setupRoutes()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/nfconfig/qos/"+"internet/"+tc.imsi, nil)
			nfServer.Router.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("expected %v, got %v", tc.expectedCode, w.Code)
			}
			var got []nfConfigApi.ImsiQos
			err := json.Unmarshal(w.Body.Bytes(), &got)
			if err != nil {
				t.Errorf("fail to unmarshal body %v", w.Body)
			}
			if !reflect.DeepEqual(got, tc.expectedData) {
				t.Errorf("expected %v, got %v", tc.expectedData, got)
			}
		})
	}
}
