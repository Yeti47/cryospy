package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeti47/cryospy/server/capture-server/handlers"
	"github.com/yeti47/cryospy/server/capture-server/middleware"
	"github.com/yeti47/cryospy/server/core/ccc/auth"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/config"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/core/notifications"
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

	// Initialize database with SQLite optimizations for concurrency
	database, err := sql.Open("sqlite3", cfg.DatabasePath+"?_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL&_cache_size=10000")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Configure connection pool for better concurrency
	database.SetMaxOpenConns(10)                  // Allow up to 10 concurrent connections
	database.SetMaxIdleConns(5)                   // Keep 5 idle connections
	database.SetConnMaxLifetime(30 * time.Minute) // Rotate connections every 30 minutes

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

	// Initialize notifiers based on configuration
	var storageNotifier notifications.StorageNotifier
	var motionNotifier notifications.MotionNotifier

	// Initialize email sender if SMTP is configured
	var emailSender notifications.EmailSender
	if cfg.SMTPSettings != nil {
		emailSender = notifications.NewSmtpSender(
			cfg.SMTPSettings.Host,
			cfg.SMTPSettings.Port,
			cfg.SMTPSettings.Username,
			cfg.SMTPSettings.Password,
			cfg.SMTPSettings.FromAddr,
		)
	} else {
		emailSender = notifications.NopSender
	}

	// Initialize storage notifier if configured
	if cfg.StorageNotificationSettings != nil && emailSender != notifications.NopSender {
		storageNotifierSettings := notifications.StorageNotificationSettings{
			Recipient:        cfg.StorageNotificationSettings.Recipient,
			MinInterval:      time.Duration(cfg.StorageNotificationSettings.MinIntervalMinutes) * time.Minute,
			WarningThreshold: cfg.StorageNotificationSettings.WarningThreshold,
		}
		storageNotifier = notifications.NewEmailStorageNotifier(storageNotifierSettings, emailSender, logger)
		logger.Info("Storage notifications enabled", "recipient", cfg.StorageNotificationSettings.Recipient)
	}

	// Initialize motion notifier if configured
	if cfg.MotionNotificationSettings != nil && emailSender != notifications.NopSender {
		motionNotifierSettings := notifications.MotionNotificationSettings{
			Recipient:   cfg.MotionNotificationSettings.Recipient,
			MinInterval: time.Duration(cfg.MotionNotificationSettings.MinIntervalMinutes) * time.Minute,
		}
		motionNotifier = notifications.NewEmailMotionNotifier(motionNotifierSettings, emailSender, logger)
		logger.Info("Motion notifications enabled", "recipient", cfg.MotionNotificationSettings.Recipient)
	}

	// Initialize auth notifier and failure tracker from unified auth event settings
	var authNotifier notifications.AuthNotifier
	var failureTracker auth.FailureTracker

	if cfg.AuthEventSettings != nil {
		// Create failure tracker settings
		autoDisableSettings := auth.AutoDisableSettings{
			Threshold:  cfg.AuthEventSettings.AutoDisableThreshold, // 0 means no auto-disable
			TimeWindow: time.Duration(cfg.AuthEventSettings.TimeWindowMinutes) * time.Minute,
		}
		failureTracker = auth.NewMemoryFailureTracker(autoDisableSettings)

		// Log what features are enabled
		if cfg.AuthEventSettings.AutoDisableThreshold > 0 {
			logger.Info("Authentication auto-disable enabled", "threshold", cfg.AuthEventSettings.AutoDisableThreshold, "timeWindow", autoDisableSettings.TimeWindow)
		}

		// Initialize auth notifier if notifications are configured
		if cfg.AuthEventSettings.NotificationRecipient != "" && cfg.AuthEventSettings.NotificationThreshold > 0 && emailSender != notifications.NopSender {
			authNotifierSettings := notifications.AuthNotificationSettings{
				Recipient:        cfg.AuthEventSettings.NotificationRecipient,
				MinInterval:      time.Duration(cfg.AuthEventSettings.MinIntervalMinutes) * time.Minute,
				FailureThreshold: cfg.AuthEventSettings.NotificationThreshold,
			}
			authNotifier = notifications.NewEmailAuthNotifier(authNotifierSettings, emailSender, logger)
			logger.Info("Authentication failure notifications enabled", "recipient", cfg.AuthEventSettings.NotificationRecipient, "threshold", cfg.AuthEventSettings.NotificationThreshold)
		} else {
			authNotifier = notifications.NopAuthNotifier
		}

		logger.Info("Authentication failure tracking enabled", "timeWindow", autoDisableSettings.TimeWindow)
	} else {
		// No auth event settings configured
		authNotifier = notifications.NopAuthNotifier
		failureTracker = auth.NopFailureTracker
	}

	storageManager := videos.NewStorageManager(logger, clipRepo, clientRepo, storageNotifier, motionNotifier)
	clipCreator := videos.NewClipCreator(
		logger,
		storageManager,
		encryptor,
		clientMekProvider,
		videoMetadataExtractor,
		thumbnailGenerator,
	)

	// Initialize handlers and middleware
	authMiddleware := middleware.NewAuthMiddleware(logger, clientVerifier, authNotifier, clientService, failureTracker)
	clipHandler := handlers.NewClipHandler(logger, clipCreator)
	clientHandler := handlers.NewClientHandler(logger, clientService)

	// Set up Gin router
	router := initializeGin(cfg)

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
