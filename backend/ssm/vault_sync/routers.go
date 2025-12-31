package vaultsync

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Route is the information for every URI.
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc gin.HandlerFunc
}

type Routes []Route

// AddSyncVaultService registers the Vault sync endpoints under /sync-vault
func AddSyncVaultService(engine *gin.Engine, middlewares ...gin.HandlerFunc) *gin.RouterGroup {
	group := engine.Group("/sync-ssm")
	if len(middlewares) > 0 {
		group.Use(middlewares...)
	}
	addRoutes(group, routes)
	return group
}

func addRoutes(group *gin.RouterGroup, routes Routes) {
	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			group.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			group.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			group.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			group.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
}

var routes = Routes{
	{
		"Sync k4 keys and users with Vault",
		http.MethodGet,
		"/sync-key",
		handleSyncKey,
	},
	{
		"Health check to k4 keys life (Vault)",
		http.MethodGet,
		"/check-k4-life",
		handleCheckK4Life,
	},
	{
		"Init the rotation for k4 manually (Vault)",
		http.MethodGet,
		"/k4-rotation",
		handleRotationKey,
	},
}
