package ssmsync

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouteStructure(t *testing.T) {
	route := Route{
		Name:    "Test Route",
		Method:  http.MethodGet,
		Pattern: "/test",
		HandlerFunc: func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		},
	}

	if route.Name != "Test Route" {
		t.Errorf("Expected Name 'Test Route', got '%s'", route.Name)
	}

	if route.Method != http.MethodGet {
		t.Errorf("Expected Method 'GET', got '%s'", route.Method)
	}

	if route.Pattern != "/test" {
		t.Errorf("Expected Pattern '/test', got '%s'", route.Pattern)
	}

	if route.HandlerFunc == nil {
		t.Error("HandlerFunc should not be nil")
	}
}

func TestRoutesSlice(t *testing.T) {
	testRoutes := Routes{
		{
			Name:    "Route 1",
			Method:  http.MethodGet,
			Pattern: "/route1",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "route1")
			},
		},
		{
			Name:    "Route 2",
			Method:  http.MethodPost,
			Pattern: "/route2",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "route2")
			},
		},
	}

	if len(testRoutes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(testRoutes))
	}
}

func TestIndexHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Index(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	expectedBody := "Hello World!"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}
}

func TestAddSyncSSMService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	group := AddSyncSSMService(engine)

	if group == nil {
		t.Error("AddSyncSSMService should return a RouterGroup")
	}
}

func TestAddSyncSSMServiceWithMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	middlewareCalled := false
	testMiddleware := func(c *gin.Context) {
		middlewareCalled = true
		c.Next()
	}

	group := AddSyncSSMService(engine, testMiddleware)

	if group == nil {
		t.Error("AddSyncSSMService should return a RouterGroup")
	}

	// Ensure variable is read to avoid unused var error
	if middlewareCalled {
		t.Error("Middleware should not be called without requests")
	}

	// We can't easily test if middleware is applied without making actual requests
	// This test just verifies that the function accepts middlewares
}

func TestRoutesDefinition(t *testing.T) {
	if len(routes) == 0 {
		t.Error("routes should not be empty")
	}

	expectedRouteCount := 3
	if len(routes) != expectedRouteCount {
		t.Errorf("Expected %d routes, got %d", expectedRouteCount, len(routes))
	}

	// Check route patterns
	patterns := make(map[string]bool)
	for _, route := range routes {
		patterns[route.Pattern] = true
	}

	expectedPatterns := []string{"/sync-key", "/check-k4-life", "/k4-rotation"}
	for _, pattern := range expectedPatterns {
		if !patterns[pattern] {
			t.Errorf("Expected route pattern '%s' not found", pattern)
		}
	}
}

func TestAddRoutesWithDifferentMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := gin.New().Group("/test")

	testRoutes := Routes{
		{
			Name:    "GET Route",
			Method:  http.MethodGet,
			Pattern: "/get",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "GET")
			},
		},
		{
			Name:    "POST Route",
			Method:  http.MethodPost,
			Pattern: "/post",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "POST")
			},
		},
		{
			Name:    "PUT Route",
			Method:  http.MethodPut,
			Pattern: "/put",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "PUT")
			},
		},
		{
			Name:    "DELETE Route",
			Method:  http.MethodDelete,
			Pattern: "/delete",
			HandlerFunc: func(c *gin.Context) {
				c.String(http.StatusOK, "DELETE")
			},
		},
	}

	addRoutes(group, testRoutes)

	// Function should not panic
}
