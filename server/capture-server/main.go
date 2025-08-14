package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/capture-server/handlers"
	"github.com/yeti47/cryospy/server/capture-server/middleware"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/config"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/core/videos"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration from default path in user's home directory
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Save the config in case it was not found or updated
	if err := cfg.SaveConfig(""); err != nil {
		log.Printf("Failed to save configuration: %v", err)
	}

	// Initialize logger
	logger := logging.CreateLogger(logging.LogLevel(cfg.LogLevel), cfg.LogPath, "capture-server")
	logger.Info("Starting capture server", "port", cfg.CapturePort)

	// Initialize database
	database, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Initialize encryption
	encryptor := encryption.NewAESEncryptor()

	// Initialize client repository and services
	clientRepo, err := clients.NewSQLiteClientRepository(database)
	if err != nil {
		log.Fatalf("Failed to create client repository: %v", err)
	}
	clientVerifier := clients.NewClientVerifier(clientRepo, encryptor)
	clientService := clients.NewClientService(logger, clientRepo, encryptor)
	clientMekProvider := clients.NewClientMekProvider(encryptor, clientRepo, clientVerifier)

	// Initialize video services
	videoMetadataExtractor := videos.NewFFmpegMetadataExtractor(logger)
	thumbnailGenerator := videos.NewFFmpegThumbnailGenerator(logger)

	clipRepo, err := videos.NewSQLiteClipRepository(database)
	if err != nil {
		log.Fatalf("Failed to create clip repository: %v", err)
	}

	storageManager := videos.NewStorageManager(logger, clipRepo, clientRepo, nil)
	clipCreator := videos.NewClipCreator(
		logger,
		storageManager,
		encryptor,
		clientMekProvider,
		videoMetadataExtractor,
		thumbnailGenerator,
	)

	// Initialize handlers and middleware
	authMiddleware := middleware.NewAuthMiddleware(logger, clientVerifier)
	clipHandler := handlers.NewClipHandler(logger, clipCreator)
	clientHandler := handlers.NewClientHandler(logger, clientService)

	// Set up Gin router
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Set up routes
	setupRoutes(router, authMiddleware, clipHandler, clientHandler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.CapturePort)
	logger.Info("Server listening", "address", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("Server failed to start", err)
		os.Exit(1)
	}
}

// setupRoutes configures the HTTP routes
func setupRoutes(router *gin.Engine, authMiddleware *middleware.AuthMiddleware, clipHandler *handlers.ClipHandler, clientHandler *handlers.ClientHandler) {
	// API routes group
	api := router.Group("/api")

	// Apply authentication middleware to all API routes
	api.Use(authMiddleware.RequireAuth())

	// Clip upload endpoint
	api.POST("/clips", clipHandler.UploadClip)

	// Client settings endpoint
	api.GET("/client/settings", clientHandler.GetClientSettings)

	// Health check endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "capture-server",
		})
	})
}
