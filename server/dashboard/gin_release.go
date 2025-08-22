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
	// Use gin.New() instead of gin.Default() to avoid debug middleware in release mode
	router := gin.New()

	// Configure trusted proxies from config
	if cfg.TrustedProxies != nil && len(cfg.TrustedProxies.Dashboard) > 0 {
		router.SetTrustedProxies(cfg.TrustedProxies.Dashboard)
	} else {
		// For dashboard in production, if no config provided, don't trust any proxies
		// This is ideal for local-only access scenarios
		router.SetTrustedProxies(nil)
	}

	return router
}
