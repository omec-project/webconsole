// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
)

type subscribersMockDB struct {
	dbadapter.DBInterface
	docs []map[string]any
}

func (m *subscribersMockDB) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	results := make([]map[string]any, 0)
	for _, doc := range m.docs {
		if matchesSubscribersFilter(doc, filter) {
			results = append(results, doc)
		}
	}
	return results, nil
}

func matchesSubscribersFilter(doc map[string]any, filter bson.M) bool {
	if len(filter) == 0 {
		return true
	}
	if andValue, ok := filter["$and"]; ok {
		switch typed := andValue.(type) {
		case []bson.M:
			for _, sub := range typed {
				if !matchesSubscribersFilter(doc, sub) {
					return false
				}
			}
			return true
		case []any:
			for _, raw := range typed {
				sub, ok := raw.(bson.M)
				if !ok {
					return false
				}
				if !matchesSubscribersFilter(doc, sub) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}

	for key, value := range filter {
		switch key {
		case "ueId":
			ue, _ := doc["ueId"].(string)
			switch v := value.(type) {
			case string:
				if ue != v {
					return false
				}
			case bson.M:
				regexStr, _ := v["$regex"].(string)
				optStr, _ := v["$options"].(string)
				if regexStr == "" {
					return false
				}
				pattern := regexStr
				if optStr == "i" {
					pattern = "(?i)" + pattern
				}
				re, err := regexp.Compile(pattern)
				if err != nil {
					return false
				}
				if !re.MatchString(ue) {
					return false
				}
			default:
				return false
			}
		case "servingPlmnId":
			plmn, _ := doc["servingPlmnId"].(string)
			want, _ := value.(string)
			if want == "" || plmn != want {
				return false
			}
		default:
			// unknown filter key
			return false
		}
	}
	return true
}

func TestGetSubscribers_LegacyArrayResponseWhenNoQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDB }()
	old := &subscribersMockDB{docs: []map[string]any{
		{"ueId": "imsi-001", "servingPlmnId": "20893"},
	}}
	dbadapter.CommonDBClient = old

	r := gin.New()
	r.GET("/api/subscriber", GetSubscribers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subscriber", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Legacy response is a JSON array.
	if len(w.Body.Bytes()) == 0 || w.Body.Bytes()[0] != '[' {
		t.Fatalf("expected JSON array response, got: %s", w.Body.String())
	}
}

func TestGetSubscribers_PaginationResponseWhenPageProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDB }()
	dbadapter.CommonDBClient = &subscribersMockDB{docs: []map[string]any{
		{"ueId": "imsi-003", "servingPlmnId": "20893"},
		{"ueId": "imsi-001", "servingPlmnId": "20893"},
		{"ueId": "imsi-002", "servingPlmnId": "20895"},
	}}

	r := gin.New()
	r.GET("/api/subscriber", GetSubscribers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/subscriber?page=1&limit=2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v; body=%s", err, w.Body.String())
	}

	items, ok := resp["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got: %T", resp["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if int(resp["total"].(float64)) != 3 {
		t.Fatalf("expected total=3, got %v", resp["total"])
	}
	if int(resp["pages"].(float64)) != 2 {
		t.Fatalf("expected pages=2, got %v", resp["pages"])
	}
}

func TestGetSubscribers_FilterAndSearchAndExact(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalDB := dbadapter.CommonDBClient
	defer func() { dbadapter.CommonDBClient = originalDB }()
	dbadapter.CommonDBClient = &subscribersMockDB{docs: []map[string]any{
		{"ueId": "imsi-2089300001", "servingPlmnId": "20893"},
		{"ueId": "imsi-2089300002", "servingPlmnId": "20893"},
		{"ueId": "imsi-001", "servingPlmnId": "20895"},
	}}

	r := gin.New()
	r.GET("/api/subscriber", GetSubscribers)

	// plmnID filter
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/subscriber?page=1&limit=50&plmnID=20893", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		items := resp["items"].([]any)
		if len(items) != 2 {
			t.Fatalf("expected 2 items for plmn filter, got %d", len(items))
		}
	}

	// q search
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/subscriber?page=1&limit=50&q=2089300002", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		items := resp["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 item for q search, got %d", len(items))
		}
	}

	// ueId exact (imsi alias)
	{
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/subscriber?page=1&limit=50&imsi=imsi-001", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		items := resp["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected 1 item for imsi exact, got %d", len(items))
		}
	}
}
