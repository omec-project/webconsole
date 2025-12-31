package configapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/dbadapter"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func TestHandleGetsK4(t *testing.T) {
	router := setupTestRouter()
	router.GET("/k4opt", HandleGetsK4)

	// Test case 1: Successful retrieval
	t.Run("Successful retrieval", func(t *testing.T) {
		mockK4Data := []map[string]any{
			{"k4": "testKey1", "k4_sno": 1},
			{"k4": "testKey2", "k4_sno": 2},
		}

		// Mock the DB call
		oldClient := dbadapter.CommonDBClient
		oldClient2 := dbadapter.AuthDBClient
		mockClient := &dbadapter.MockDBClient{
			GetManyFn: func(collName string, filter bson.M) ([]map[string]any, error) {
				return mockK4Data, nil
			},
		}
		dbadapter.CommonDBClient = mockClient
		dbadapter.AuthDBClient = mockClient
		defer func() {
			dbadapter.CommonDBClient = oldClient
			dbadapter.AuthDBClient = oldClient2
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/k4opt", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.K4
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
	})

	// Test case 2: Database error
	t.Run("Database error", func(t *testing.T) {
		// Mock the DB call with error
		oldClient := dbadapter.CommonDBClient
		oldClient2 := dbadapter.AuthDBClient
		mockClient := &dbadapter.MockDBClient{
			GetManyFn: func(collName string, filter bson.M) ([]map[string]any, error) {
				return nil, assert.AnError
			},
		}
		dbadapter.CommonDBClient = mockClient
		dbadapter.AuthDBClient = mockClient
		defer func() {
			dbadapter.CommonDBClient = oldClient
			dbadapter.AuthDBClient = oldClient2
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/k4opt", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandleGetK4(t *testing.T) {
	router := setupTestRouter()
	router.GET("/k4opt/:idsno", HandleGetK4)

	// Test case 1: Successful retrieval
	t.Run("Successful retrieval", func(t *testing.T) {
		mockK4Data := map[string]any{
			"k4":     "testKey1",
			"k4_sno": int32(1),
		}

		// Mock the DB call
		oldClient := dbadapter.AuthDBClient
		dbadapter.AuthDBClient = &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return mockK4Data, nil
			},
		}
		defer func() { dbadapter.AuthDBClient = oldClient }()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/k4opt/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test case 2: Database error
	t.Run("Database error", func(t *testing.T) {
		// Mock the DB call with error
		oldClient := dbadapter.AuthDBClient
		dbadapter.AuthDBClient = &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return nil, assert.AnError
			},
		}
		defer func() { dbadapter.AuthDBClient = oldClient }()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/k4opt/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandlePostK4(t *testing.T) {
	router := setupTestRouter()
	router.POST("/k4opt", HandlePostK4)

	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
		},
	}
	// Test case 1: Successful post
	t.Run("Successful post", func(t *testing.T) {
		k4Data := models.K4{
			K4:     "1234ABCDEF",
			K4_SNO: byte(1),
		}
		jsonData, err := json.Marshal(k4Data)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		// Mock the DB calls
		oldAuthClient := dbadapter.AuthDBClient
		oldCommonClient := dbadapter.CommonDBClient

		mockClient := &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return nil, assert.AnError
			},
			PostFn: func(collName string, filter bson.M, postData map[string]any) (bool, error) {
				return true, nil
			},
			PutOneFn: func(collName string, filter bson.M, putData map[string]any) (bool, error) {
				return true, nil
			},
		}

		dbadapter.AuthDBClient = mockClient
		dbadapter.CommonDBClient = mockClient

		defer func() {
			dbadapter.AuthDBClient = oldAuthClient
			dbadapter.CommonDBClient = oldCommonClient
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/k4opt", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json") // Añadido header Content-Type
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		if w.Code != http.StatusCreated {
			t.Logf("Response body: %s", w.Body.String()) // Para debug
		}
	})

	// Test case 2: Invalid JSON
	t.Run("Invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/k4opt", bytes.NewBuffer([]byte("invalid json")))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlePutK4(t *testing.T) {
	router := setupTestRouter()
	router.PUT("/k4opt/:idsno", HandlePutK4)

	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
		},
	}

	// Test case 1: Successful update
	t.Run("Successful update", func(t *testing.T) {
		k4Data := models.K4{
			K4:     "1234ABCDEF",
			K4_SNO: byte(1),
		}
		jsonData, err := json.Marshal(k4Data)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		// Mock the DB calls
		oldAuthClient := dbadapter.AuthDBClient
		oldCommonClient := dbadapter.CommonDBClient

		mockClient := &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return map[string]any{"k4": "1234ABCDEF", "k4_sno": "1"}, nil
			},
			PutOneFn: func(collName string, filter bson.M, putData map[string]any) (bool, error) {
				return true, nil
			},
		}

		dbadapter.AuthDBClient = mockClient
		dbadapter.CommonDBClient = mockClient

		defer func() {
			dbadapter.AuthDBClient = oldAuthClient
			dbadapter.CommonDBClient = oldCommonClient
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", "/k4opt/1", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test case 2: K4 not found
	t.Run("K4 not found", func(t *testing.T) {
		k4Data := models.K4{
			K4:     "1234ABCDEF",
			K4_SNO: byte(1),
		}
		jsonData, err := json.Marshal(k4Data)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		// Mock the DB calls
		oldClient := dbadapter.AuthDBClient
		oldCommonClient := dbadapter.CommonDBClient
		dbadapter.AuthDBClient = &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return nil, nil
			},
		}
		dbadapter.CommonDBClient = &dbadapter.MockDBClient{
			PutOneFn: func(collName string, filter bson.M, data map[string]any) (bool, error) {
				return false, errors.New("K4 not found")
			},
		}
		defer func() {
			dbadapter.AuthDBClient = oldClient
			dbadapter.CommonDBClient = oldCommonClient
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("PUT", "/k4opt/1", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandleDeleteK4(t *testing.T) {
	router := setupTestRouter()
	router.DELETE("/k4opt/:idsno", HandleDeleteK4)

	oldConfig := factory.WebUIConfig
	defer func() { factory.WebUIConfig = oldConfig }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SSM: &factory.SSM{
				AllowSsm: false,
			},
			Vault: &factory.Vault{
				AllowVault: false,
			},
		},
	}
	// Test case 1: Successful deletion
	t.Run("Successful deletion", func(t *testing.T) {
		// Mock the DB calls
		oldAuthClient := dbadapter.AuthDBClient
		oldCommonClient := dbadapter.CommonDBClient

		mockClient := &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return map[string]any{"k4": "1234ABCDEF", "k4_sno": "1"}, nil
			},
			DeleteOneFn: func(collName string, filter bson.M) error {
				return nil
			},
		}

		dbadapter.AuthDBClient = mockClient
		dbadapter.CommonDBClient = mockClient

		defer func() {
			dbadapter.AuthDBClient = oldAuthClient
			dbadapter.CommonDBClient = oldCommonClient
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("DELETE", "/k4opt/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test case 2: K4 not found
	t.Run("K4 not found", func(t *testing.T) {
		// Mock the DB calls
		oldClient := dbadapter.AuthDBClient
		oldCommonClient := dbadapter.CommonDBClient
		dbadapter.AuthDBClient = &dbadapter.MockDBClient{
			GetOneFn: func(collName string, filter bson.M) (map[string]any, error) {
				return nil, nil
			},
		}
		dbadapter.CommonDBClient = &dbadapter.MockDBClient{
			DeleteOneFn: func(collName string, filter bson.M) error {
				return errors.New("K4 not found")
			},
		}
		defer func() {
			dbadapter.AuthDBClient = oldClient
			dbadapter.CommonDBClient = oldCommonClient
		}()

		w := httptest.NewRecorder()
		req, err := http.NewRequest("DELETE", "/k4opt/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
