//go:build !release
// +build !release

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/core/config"
)

// initializeGin sets up Gin in debug mode for development builds
func initializeGin(_ *config.Config) *gin.Engine {
	// Gin will be in debug mode by default
	router := gin.New()

	// For development builds, trust all proxies (don't restrict)
	// This matches the original Gin default behavior for development ease

	return router
}
