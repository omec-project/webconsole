//go:build ui
// +build ui

package webui_service

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed webui/frontend/dist/*
var staticFiles embed.FS

func (*WEBUI) SetUpStaticFiles(router *gin.Engine) {
	router.StaticFS("/static", http.FS(staticFiles))
}
