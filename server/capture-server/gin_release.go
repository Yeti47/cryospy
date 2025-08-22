//go:build release
// +build release

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/config"
)

// initializeGin sets up Gin in release mode for production builds
func initializeGin(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Configure trusted proxies from config
	if cfg.TrustedProxies != nil && len(cfg.TrustedProxies.CaptureServer) > 0 {
		router.SetTrustedProxies(cfg.TrustedProxies.CaptureServer)
	} else {
		// For production, if no config provided, don't trust any proxies
		// This is the most secure default
		router.SetTrustedProxies(nil)
	}

	return router
}
