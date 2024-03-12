//go:build !ui
// +build !ui

package webui_service

import "github.com/gin-gonic/gin"

func (*WEBUI) SetUpStaticFiles(router *gin.Engine) {}
