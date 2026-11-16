// Copyright (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nfconfig

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/openapi/v2/nfconfigpoll"
	"github.com/omec-project/webconsole/configmodels"
)

// The network functions poll these routes and decode them with nfconfigpoll.FetchJSON into the types of
// openapi/nfConfigApi (and, for the PLMN list of the UDM, AUSF, UDR and NSSF, openapi/models). The tests here go
// the whole way: the in-memory configuration is built by the sync functions from webconsole data, served by the
// real routes with the middleware of NewNFConfigServer, fetched with the NFs' own fetch function, and compared
// with what was served. A change of the API that the pollers cannot decode, or that they would take for a change
// of configuration, fails here instead of in a deployment.

const (
	contractImsDnn = "ims"
	contractSdA    = "010203"
	contractSdB    = "112233"

	// the routes the NFs poll, and the fixtures of the deployment the tests serve
	routePlmn              = "/nfconfig/plmn"
	routePlmnSnssai        = "/nfconfig/plmn-snssai"
	routeAccessMobility    = "/nfconfig/access-mobility"
	routeSessionManagement = "/nfconfig/session-management"
	routePolicyControl     = "/nfconfig/policy-control"

	contractDgInternet = "dg-internet"
	contractDgIms      = "dg-ims"
	contractImsiIms    = "001010123456790"
	contractImsPool    = "10.2.2.0/24"
)

// contractServer serves cfg the way NewNFConfigServer does, without the database that it syncs from.
func contractServer(t *testing.T, cfg inMemoryConfig) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(enforceAcceptJSON())
	n := &NFConfigServer{Router: router, inMemoryConfig: cfg}
	n.setupRoutes()
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// contractConfig is a deployment with two slices on two PLMNs, an IMS DNN next to the internet one, and
// subscribers in both device groups.
func contractConfig() inMemoryConfig {
	slices := prepareMultipleSlices([]networkSliceParams{
		{
			sliceName: "slice-a", mcc: "001", mnc: "01", sst: "1", sd: contractSdA,
			deviceGroups: []string{contractDgInternet, contractDgIms}, upfHostname: "upf.local", upfPort: "8805", gnbNames: []string{"gnb-1", "gnb-2"},
		},
		{
			sliceName: "slice-b", mcc: "208", mnc: "93", sst: "2", sd: contractSdB,
			deviceGroups: []string{contractDgInternet}, upfHostname: "upf-b.local", upfPort: "8805", gnbNames: []string{"gnb-3"},
		},
	})
	deviceGroups := map[string]configmodels.DeviceGroups{}
	for _, p := range []deviceGroupParams{
		{name: contractDgInternet, imsis: []string{"001010123456789", "208930123456789"}, dnn: dnnInternet, dnsPrimary: dnsPrimaryTest, ueIpPool: ueIpPoolTest, mtu: 1500},
		{name: contractDgIms, imsis: []string{contractImsiIms}, dnn: contractImsDnn, dnsPrimary: dnsPrimaryTest, pcscfPrimary: "10.10.10.10", ueIpPool: contractImsPool, mtu: 1400},
	} {
		name, dg := makeDeviceGroup(p)
		deviceGroups[name] = dg
	}

	var cfg inMemoryConfig
	cfg.syncPlmn(slices)
	cfg.syncPlmnSnssai(slices)
	cfg.syncAccessAndMobility(slices)
	cfg.syncSessionManagement(slices, deviceGroups)
	cfg.syncPolicyControl(slices, deviceGroups)
	return cfg
}

// emptyContractConfig is what the sync functions make of a deployment without slices.
func emptyContractConfig() inMemoryConfig {
	var cfg inMemoryConfig
	cfg.syncPlmn(nil)
	cfg.syncPlmnSnssai(nil)
	cfg.syncAccessAndMobility(nil)
	cfg.syncSessionManagement(nil, map[string]configmodels.DeviceGroups{})
	cfg.syncPolicyControl(nil, map[string]configmodels.DeviceGroups{})
	return cfg
}

// fetch is what the NFs' pollers do: GET with the Accept and Content-Type rules of nfconfigpoll.FetchJSON.
func fetch[T any](t *testing.T, srv *httptest.Server, path string) T {
	t.Helper()
	got, err := nfconfigpoll.FetchJSON[T](context.Background(), srv.Client(), srv.URL+path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return got
}

// normalized is the form in which the AMF and SMF pollers compare configurations (nfconfigpoll.JSONRoundTrip).
func normalized[T any](t *testing.T, v T) T {
	t.Helper()
	out, err := nfconfigpoll.JSONRoundTrip(v)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	return out
}

func requireSameConfig[T any](t *testing.T, served, fetched T) {
	t.Helper()
	if want, got := normalized(t, served), normalized(t, fetched); !reflect.DeepEqual(want, got) {
		t.Errorf("the NF does not get what was served:\n served  %+v\n fetched %+v", want, got)
	}
}

func TestPollersDecodeWhatTheWebconsoleServes(t *testing.T) {
	cfg := contractConfig()
	srv := contractServer(t, cfg)

	t.Run("plmn into nfConfigApi.PlmnId (NRF, and the generated API)", func(t *testing.T) {
		got := fetch[[]nfConfigApi.PlmnId](t, srv, routePlmn)
		if len(got) != 2 || got[0].GetMcc() != "001" || got[1].GetMcc() != "208" {
			t.Errorf("plmn = %+v, want 001/01 and 208/93 in order", got)
		}
		requireSameConfig(t, cfg.plmn, got)
	})

	// UDM, AUSF, UDR, NRF and the registration of the NSSF poll the PLMN list into the type of openapi/models.
	t.Run("plmn into models.PlmnId (UDM, AUSF, UDR)", func(t *testing.T) {
		got := fetch[[]models.PlmnId](t, srv, routePlmn)
		if len(got) != len(cfg.plmn) {
			t.Fatalf("%d PLMNs, served %d", len(got), len(cfg.plmn))
		}
		for i, p := range cfg.plmn {
			if got[i].Mcc != p.GetMcc() || got[i].Mnc != p.GetMnc() {
				t.Errorf("PLMN %d = %s/%s, served %s/%s", i, got[i].Mcc, got[i].Mnc, p.GetMcc(), p.GetMnc())
			}
		}
	})

	t.Run("plmn-snssai (NSSF)", func(t *testing.T) {
		got := fetch[[]nfConfigApi.PlmnSnssai](t, srv, routePlmnSnssai)
		if len(got) != 2 {
			t.Fatalf("%d entries, want one per PLMN", len(got))
		}
		requireSameConfig(t, cfg.plmnSnssai, got)
	})

	t.Run("access-mobility (AMF)", func(t *testing.T) {
		got := fetch[[]nfConfigApi.AccessAndMobility](t, srv, routeAccessMobility)
		if len(got) == 0 {
			t.Fatal("no access and mobility configuration")
		}
		requireSameConfig(t, cfg.accessAndMobility, got)
		// the AMF takes the polled value through the JSON round trip and compares it with the next poll: a
		// second round trip must not change it, or every poll would look like a new configuration
		if once := normalized(t, got); !reflect.DeepEqual(once, normalized(t, once)) {
			t.Error("the polled configuration is not stable under the AMF's normalization")
		}
	})

	t.Run("session-management (SMF)", func(t *testing.T) {
		got := fetch[[]nfConfigApi.SessionManagement](t, srv, routeSessionManagement)
		if len(got) == 0 {
			t.Fatal("no session management configuration")
		}
		requireSameConfig(t, cfg.sessionManagement, got)
		if once := normalized(t, got); !reflect.DeepEqual(once, normalized(t, once)) {
			t.Error("the polled configuration is not stable under the SMF's normalization")
		}
		dnns := map[string]bool{}
		for _, s := range got {
			for _, d := range s.IpDomain {
				dnns[d.GetDnnName()] = true
			}
		}
		if !dnns[dnnInternet] || !dnns[contractImsDnn] {
			t.Errorf("DNNs %v, want internet and ims", dnns)
		}
	})

	t.Run("policy-control (PCF)", func(t *testing.T) {
		got := fetch[[]nfConfigApi.PolicyControl](t, srv, routePolicyControl)
		if len(got) == 0 {
			t.Fatal("no policy control configuration")
		}
		requireSameConfig(t, cfg.policyControl, got)
		for _, p := range got {
			if len(p.PccRules) == 0 {
				t.Errorf("slice %s/%s has no PCC rules", p.PlmnId.GetMcc(), p.PlmnId.GetMnc())
			}
		}
	})
}

// The pollers start from an empty, non-nil configuration and only act on a difference. A deployment without
// slices has to be served as an empty list: null decodes to a nil slice, which they would take for a change,
// and the NF would deregister or re-register on its first poll.
func TestEmptyConfigurationIsServedAsEmptyLists(t *testing.T) {
	srv := contractServer(t, emptyContractConfig())

	for _, path := range []string{routePlmn, routePlmnSnssai, routeAccessMobility, routeSessionManagement, routePolicyControl} {
		t.Run(path, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Accept", "application/json")
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "[]" {
				t.Errorf("body %q, want %q", body, "[]")
			}
		})
	}

	// the same through the NFs' types: not nil, so that it equals what they start from
	if got := fetch[[]models.PlmnId](t, srv, routePlmn); got == nil || len(got) != 0 {
		t.Errorf("plmn = %#v, want an empty non-nil list", got)
	}
	if got := fetch[[]nfConfigApi.SessionManagement](t, srv, routeSessionManagement); got == nil || len(got) != 0 {
		t.Errorf("session-management = %#v, want an empty non-nil list", got)
	}
	if got := fetch[[]nfConfigApi.AccessAndMobility](t, srv, routeAccessMobility); got == nil || len(got) != 0 {
		t.Errorf("access-mobility = %#v, want an empty non-nil list", got)
	}
}

// The pollers send Accept: application/json and nothing else, and the routes insist on it. Both ends rely on the
// same string, so this pins the request the pollers make and the answer when they do not make it.
func TestRoutesRequireTheAcceptHeaderThePollersSend(t *testing.T) {
	srv := contractServer(t, contractConfig())
	do := func(accept string) (int, string) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+routePlmn, nil)
		if err != nil {
			t.Fatal(err)
		}
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, resp.Header.Get("Content-Type")
	}

	if code, ct := do("application/json"); code != http.StatusOK || ct == "" {
		t.Errorf("Accept application/json: status %d, Content-Type %q", code, ct)
	}
	if code, _ := do("text/html"); code != http.StatusBadRequest {
		t.Errorf("Accept text/html: status %d, want 400", code)
	}
	if code, _ := do(""); code != http.StatusBadRequest {
		t.Errorf("no Accept: status %d, want 400", code)
	}

	// and what FetchJSON sends is accepted
	var served []nfConfigApi.PlmnId
	raw, err := json.Marshal(fetch[[]nfConfigApi.PlmnId](t, srv, routePlmn))
	if err != nil || json.Unmarshal(raw, &served) != nil || len(served) == 0 {
		t.Errorf("the poller's request was not answered with a configuration: %s (%v)", raw, err)
	}
}
