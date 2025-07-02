// SPDX-FileCopyrightText: 2025 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package nfconfig

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/nfConfigApi"
	"github.com/omec-project/util/logger"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type MockDBClient struct {
	dbadapter.DBInterface
	Slices []configmodels.Slice
	err    error
}

func (m *MockDBClient) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	for _, s := range m.Slices {
		ns := configmodels.ToBsonM(s)
		if ns == nil {
			panic("failed to convert network slice to BsonM")
		}
		results = append(results, ns)
	}
	return results, m.err
}

func makeNetworkSlice(mcc, mnc, sst string, sd string, tacs []int32) configmodels.Slice {
	plmnId := configmodels.SliceSiteInfoPlmn{
		Mcc: mcc,
		Mnc: mnc,
	}
	siteInfo := configmodels.SliceSiteInfo{
		SiteName: "test",
		Plmn:     plmnId,
		GNodeBs:  []configmodels.SliceSiteInfoGNodeBs{},
	}
	for _, tac := range tacs {
		gNodeB := configmodels.SliceSiteInfoGNodeBs{
			Name: fmt.Sprintf("test-gnb-%d", tac),
			Tac:  tac,
		}
		siteInfo.GNodeBs = append(siteInfo.GNodeBs, gNodeB)
	}
	sliceId := configmodels.SliceSliceId{
		Sst: sst,
		Sd:  sd,
	}
	networkSlice := configmodels.Slice{
		SliceName: "slice1",
		SliceId:   sliceId,
		SiteInfo:  siteInfo,
	}
	return networkSlice
}

func makeSnssaiWithSd(sst int32, sd string) nfConfigApi.Snssai {
	s := nfConfigApi.NewSnssai(sst)
	s.SetSd(sd)
	return *s
}

func TestNewNFConfig_nil_config(t *testing.T) {
	_, err := NewNFConfigServer(nil)
	if err == nil {
		t.Errorf("expected error for nil config, got nil.")
	}
}

func TestNewNFConfig_various_configs(t *testing.T) {
	testCases := []struct {
		name   string
		config *factory.Config
	}{
		{
			name: "correct TLS configuration and warn log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "warn",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "correct TLS configuration and info log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "info",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "missing key and error log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "error",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
					},
				},
			},
		},
		{
			name: "missing pem and debug log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "debug",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "invalid debug level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "invalid_level",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
		{
			name: "correct TLS configuration and wrong log level",
			config: &factory.Config{
				Logger: &logger.Logger{
					WEBUI: &logger.LogSetting{
						DebugLevel: "invalid",
					},
				},
				Configuration: &factory.Configuration{
					NfConfigTLS: &factory.TLS{
						PEM: "test.pem",
						Key: "test.key",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{},
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			nf, err := NewNFConfigServer(tc.config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if nf == nil {
				t.Errorf("expected non-nil NFConfigInterface, got nil")
			}
		})
	}
}

func TestNFConfigRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockValidConfig := &factory.Config{
		Logger: &logger.Logger{
			WEBUI: &logger.LogSetting{
				DebugLevel: "debug",
			},
		},
		Configuration: &factory.Configuration{
			NfConfigTLS: &factory.TLS{
				PEM: "test.pem",
				Key: "test.key",
			},
		},
	}

	mockDB := &MockDBClient{
		Slices: []configmodels.Slice{},
	}
	originalDBClient := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDBClient }()
	dbadapter.CommonDBClient = mockDB

	nfInterface, err := NewNFConfigServer(mockValidConfig)
	if err != nil {
		t.Fatalf("failed to initialize NFConfig: %v", err)
	}

	nf, ok := nfInterface.(*NFConfigServer)
	if !ok {
		t.Fatalf("expected *NFConfig type")
	}

	testCases := []struct {
		name         string
		path         string
		acceptHeader string
		wantStatus   int
	}{
		{
			name:         "access mobility endpoint status OK",
			path:         "/nfconfig/access-mobility",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "plmn endpoint status OK",
			path:         "/nfconfig/plmn",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "plmn-snssai endpoint status OK",
			path:         "/nfconfig/plmn-snssai",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "policy control endpoint status OK",
			path:         "/nfconfig/policy-control",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "session management endpoint status OK",
			path:         "/nfconfig/session-management",
			acceptHeader: "application/json",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "access mobility endpoint invalid accept header",
			path:         "/nfconfig/access-mobility",
			acceptHeader: "",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "plmn endpoint invalid accept header",
			path:         "/nfconfig/plmn",
			acceptHeader: "json",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "plmn-snssai endpoint invalid accept header",
			path:         "/nfconfig/plmn-snssai",
			acceptHeader: "text/html",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "policy control endpoint invalid accept header",
			path:         "/nfconfig/policy-control",
			acceptHeader: "text/html",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "session management endpoint invalid accept header",
			path:         "/nfconfig/session-management",
			acceptHeader: "application/jsons",
			wantStatus:   http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			req.Header.Set("Accept", tc.acceptHeader)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			nf.router().ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestNFConfigStart(t *testing.T) {
	tests := []struct {
		name    string
		config  *factory.Configuration
		wantErr bool
	}{
		{
			name: "HTTP server start and graceful shutdown",
			config: &factory.Configuration{
				NfConfigTLS: nil,
			},
			wantErr: false,
		},
		{
			name: "HTTPS server start and graceful shutdown",
			config: &factory.Configuration{
				NfConfigTLS: &factory.TLS{
					PEM: "testdata/test.pem",
					Key: "testdata/test.key",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Starting test: %s", tt.name)
			gin.SetMode(gin.TestMode)
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{},
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			nfconf := &NFConfigServer{
				config: tt.config,
				Router: gin.New(),
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			errChan := make(chan error, 1)
			syncChan := make(chan struct{}, 1)
			go func() {
				t.Logf("starting server")
				err := nfconf.Start(ctx, syncChan)
				t.Logf("server stopped with error: %v", err)
				errChan <- err
			}()

			time.Sleep(500 * time.Millisecond)
			t.Logf("triggering shutdown")
			cancel()

			select {
			case err := <-errChan:
				if tt.wantErr && err == nil {
					t.Errorf("got error = nil, wantErr %v", tt.wantErr)
				}
				if !tt.wantErr && err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("test timed out waiting for server to stop")
			}
		})
	}
}

func TestNFConfig_Start_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc1 := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	nfc2 := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	errChan := make(chan error, 1)
	syncChan := make(chan struct{}, 1)
	go func() {
		errChan <- nfc1.Start(ctx1, syncChan)
	}()
	time.Sleep(10 * time.Millisecond)

	ctx2 := context.Background()
	err := nfc2.Start(ctx2, syncChan)
	if err == nil {
		t.Error("expected error when starting server on same port, got nil")
	}
	cancel1()
	<-errChan
}

func TestNFConfig_Start_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nfc := &NFConfigServer{
		config: &factory.Configuration{},
		Router: gin.New(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	syncChan := make(chan struct{}, 1)
	go func() {
		errChan <- nfc.Start(ctx, syncChan)
	}()
	time.Sleep(100 * time.Millisecond)
	t.Logf("triggering context cancellation")
	cancel()
	err := <-errChan
	if err != nil {
		t.Errorf("got error = %v, want nil after context cancellation", err)
	}
}

func TestSyncWithRetry_Success_CallsSyncInMemoryConfig(t *testing.T) {
	n := &NFConfigServer{}

	called := false
	originalSyncInMemoryFunc := syncInMemoryConfigFunc
	defer func() { syncInMemoryConfigFunc = originalSyncInMemoryFunc }()
	syncInMemoryConfigFunc = func(n *NFConfigServer) error {
		called = true
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.syncWithRetry(ctx)
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Fatal("expected syncInMemoryConfig to be called")
	}
	cancel()
}

func TestSyncWithRetry_RetryInCaseOfFailure(t *testing.T) {
	n := &NFConfigServer{}

	callCount := 0
	originalSyncInMemoryFunc := syncInMemoryConfigFunc
	defer func() { syncInMemoryConfigFunc = originalSyncInMemoryFunc }()
	syncInMemoryConfigFunc = func(n *NFConfigServer) error {
		callCount++
		if callCount < 3 {
			return fmt.Errorf("mock error")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.syncWithRetry(ctx)

	time.Sleep(10 * time.Second)
	if callCount != 3 {
		t.Fatalf("expected 3 calls to syncInMemoryConfigFunc, got %d", callCount)
	}
	cancel()
}

func TestSyncInMemoryConfig_Success(t *testing.T) {
	tests := []struct {
		name                      string
		slices                    []configmodels.Slice
		expectedPlmn              []nfConfigApi.PlmnId
		expectedPlmnSnssai        []nfConfigApi.PlmnSnssai
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name: "Two slices same PLMN different S-NSSAI",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "2", "abcd", []int32{1}),
				makeNetworkSlice("123", "23", "1", "01234", []int32{2}),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"2"},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(2, "abcd"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "Two slices same PLMN duplicate S-NSSAI",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234", []int32{1}),
				makeNetworkSlice("123", "23", "1", "01234", []int32{2}),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1", "2"},
				},
			},
		},
		{
			name: "Several slices different PLMN are ordered",
			slices: []configmodels.Slice{
				makeNetworkSlice("999", "455", "2", "abcd", []int32{1}),
				makeNetworkSlice("123", "23", "3", "3333", []int32{1}),
				makeNetworkSlice("999", "455", "2", "", []int32{1}),
				makeNetworkSlice("123", "23", "3", "123", []int32{1}),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
				*nfConfigApi.NewPlmnId("999", "455"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(3, "123"),
						makeSnssaiWithSd(3, "3333"),
					},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("999", "455"),
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(2),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(3, "123"),
					Tacs:   []string{"1"},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(3, "3333"),
					Tacs:   []string{"1"},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("999", "455"),
					Snssai: *nfConfigApi.NewSnssai(2),
					Tacs:   []string{"1"},
				},
				{
					PlmnId: *nfConfigApi.NewPlmnId("999", "455"),
					Snssai: makeSnssaiWithSd(2, "abcd"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "One slice no SD",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "", []int32{1}),
			},
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("123", "23"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						*nfConfigApi.NewSnssai(1),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: *nfConfigApi.NewSnssai(1),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name:                      "Empty slices",
			slices:                    []configmodels.Slice{},
			expectedPlmn:              []nfConfigApi.PlmnId{},
			expectedPlmnSnssai:        []nfConfigApi.PlmnSnssai{},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: tc.slices,
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{},
			}

			err := n.syncInMemoryConfig()
			if err != nil {
				t.Errorf("expected no error. Got %s", err)
			}
			if !reflect.DeepEqual(tc.expectedPlmn, n.inMemoryConfig.plmn) {
				t.Errorf("Expected PLMN %v, got %v", tc.expectedPlmn, n.inMemoryConfig.plmn)
			}
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai)
			}
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility) {
				t.Errorf("Expected Access and Mobility %v, got %v", tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility)
			}
		})
	}
}

func TestSyncInMemoryConfig_DBError_KeepsPreviousConfig(t *testing.T) {
	tests := []struct {
		name                      string
		expectedPlmn              []nfConfigApi.PlmnId
		expectedPlmnSnssai        []nfConfigApi.PlmnSnssai
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name:                      "Initial empty PLMN S-NSSAI config",
			expectedPlmn:              []nfConfigApi.PlmnId{},
			expectedPlmnSnssai:        []nfConfigApi.PlmnSnssai{},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{},
		},
		{
			name: "Initial not empty PLMN S-NSSAI config",
			expectedPlmn: []nfConfigApi.PlmnId{
				*nfConfigApi.NewPlmnId("44", "22"),
				*nfConfigApi.NewPlmnId("167", "24"),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
						makeSnssaiWithSd(2, "abcd"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1", "2"},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: []configmodels.Slice{makeNetworkSlice("999", "99", "9", "999", []int32{1})},
				err:    fmt.Errorf("mock error"),
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{
					plmn:              tc.expectedPlmn,
					plmnSnssai:        tc.expectedPlmnSnssai,
					accessAndMobility: tc.expectedAccessAndMobility,
				},
			}

			err := n.syncInMemoryConfig()

			if err == nil {
				t.Errorf("expected error. Got nil")
			}
			if !reflect.DeepEqual(tc.expectedPlmn, n.inMemoryConfig.plmn) {
				t.Errorf("Expected PLMN %v, got %v", tc.expectedPlmn, n.inMemoryConfig.plmn)
			}
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai)
			}
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility) {
				t.Errorf("Expected Access and Mobility %v, got %v", tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility)
			}
		})
	}
}

func TestSyncInMemoryConfig_UnmarshalError_IgnoresNetworkSlice(t *testing.T) {
	tests := []struct {
		name                      string
		slices                    []configmodels.Slice
		expectedPlmnSnssai        []nfConfigApi.PlmnSnssai
		expectedAccessAndMobility []nfConfigApi.AccessAndMobility
	}{
		{
			name: "Invalid SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234", []int32{1}),
				makeNetworkSlice("123", "455", "a", "56789", []int32{1}),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "Empty SST is ignored",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "23", "1", "01234", []int32{1}),
				makeNetworkSlice("123", "455", "", "56789", []int32{1}),
			},
			expectedPlmnSnssai: []nfConfigApi.PlmnSnssai{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					SNssaiList: []nfConfigApi.Snssai{
						makeSnssaiWithSd(1, "01234"),
					},
				},
			},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{
				{
					PlmnId: *nfConfigApi.NewPlmnId("123", "23"),
					Snssai: makeSnssaiWithSd(1, "01234"),
					Tacs:   []string{"1"},
				},
			},
		},
		{
			name: "Invalid SST final list is empty",
			slices: []configmodels.Slice{
				makeNetworkSlice("123", "455", "a", "56789", []int32{1}),
			},
			expectedPlmnSnssai:        []nfConfigApi.PlmnSnssai{},
			expectedAccessAndMobility: []nfConfigApi.AccessAndMobility{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDBClient{
				Slices: tc.slices,
			}
			originalDBClient := dbadapter.CommonDBClient
			defer func() { dbadapter.CommonDBClient = originalDBClient }()
			dbadapter.CommonDBClient = mockDB
			n := &NFConfigServer{
				inMemoryConfig: inMemoryConfig{},
			}

			err := n.syncInMemoryConfig()
			if err != nil {
				t.Errorf("expected no error. Got %s", err)
			}
			if !reflect.DeepEqual(tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai) {
				t.Errorf("Expected PLMN-SNSSAI %v, got %v", tc.expectedPlmnSnssai, n.inMemoryConfig.plmnSnssai)
			}
			if !reflect.DeepEqual(tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility) {
				t.Errorf("Expected Access and Mobility %v, got %v", tc.expectedAccessAndMobility, n.inMemoryConfig.accessAndMobility)
			}
		})
	}
}
