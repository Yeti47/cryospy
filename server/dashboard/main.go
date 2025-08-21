package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/config"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/core/streaming"
	"github.com/yeti47/cryospy/server/core/videos"
	dashboard_sessions "github.com/yeti47/cryospy/server/dashboard/sessions"
	"github.com/yeti47/cryospy/server/dashboard/web/handlers"
	"github.com/yeti47/cryospy/server/dashboard/web/middleware"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// try to save the config in case it was not found
	if err := cfg.SaveConfig(""); err != nil {
		log.Printf("Failed to save configuration: %v", err)
	}

	// Set up logger
	logger := logging.CreateLogger(logging.LogLevel(cfg.LogLevel), cfg.LogPath, "dashboard")

	// Set up database connection with SQLite optimizations for concurrency
	dbConn, err := sql.Open("sqlite3", cfg.DatabasePath+"?_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL&_cache_size=10000")
	if err != nil {
		logger.Error("Failed to open database", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Configure connection pool for better concurrency
	dbConn.SetMaxOpenConns(10)                  // Allow up to 10 concurrent connections
	dbConn.SetMaxIdleConns(5)                   // Keep 5 idle connections
	dbConn.SetConnMaxLifetime(30 * time.Minute) // Rotate connections every 30 minutes

	// Set up repositories
	mekRepo, err := encryption.NewSQLiteMekRepository(dbConn)
	if err != nil {
		logger.Error("Failed to create MEK repository", err)
		os.Exit(1)
	}
	clientRepo, err := clients.NewSQLiteClientRepository(dbConn)
	if err != nil {
		logger.Error("Failed to create client repository", err)
		os.Exit(1)
	}
	clipRepo, err := videos.NewSQLiteClipRepository(dbConn)
	if err != nil {
		logger.Error("Failed to create clip repository", err)
		os.Exit(1)
	}

	// Set up services
	encryptor := encryption.NewAESEncryptor()
	mekService := encryption.NewMekService(logger, mekRepo, encryptor)
	clientService := clients.NewClientService(logger, clientRepo, encryptor)
	clipReader := videos.NewClipReader(logger, clipRepo, encryptor)
	clipDeleter := videos.NewClipDeleter(logger, clipRepo)
	storageManager := videos.NewStorageManager(logger, clipRepo, clientRepo, nil, nil)

	// Set up streaming services
	normalizationSettings := streaming.DefaultNormalizationSettings()
	if cfg.StreamingSettings != nil {
		normalizationSettings.Width = cfg.StreamingSettings.Width
		normalizationSettings.Height = cfg.StreamingSettings.Height
		normalizationSettings.VideoBitrate = cfg.StreamingSettings.VideoBitrate
		normalizationSettings.VideoCodec = cfg.StreamingSettings.VideoCodec
		normalizationSettings.FrameRate = cfg.StreamingSettings.FrameRate
	}
	clipNormalizer := streaming.NewFFmpegClipNormalizer(logger, "", normalizationSettings)

	// Set up caching if enabled
	var cachedNormalizer streaming.ClipNormalizer = clipNormalizer
	if cfg.StreamingSettings != nil && cfg.StreamingSettings.Cache.Enabled {
		cache := streaming.NewNormalizedClipCache(cfg.StreamingSettings.Cache.MaxSizeBytes, logger)
		cachedNormalizer = streaming.NewCachedClipNormalizer(clipNormalizer, cache, logger)
	}

	playlistGenerator := streaming.NewM3U8PlaylistGenerator(logger)
	streamingConfig := config.StreamingSettings{LookAhead: 10}
	if cfg.StreamingSettings != nil {
		streamingConfig = *cfg.StreamingSettings
	}
	streamingService := streaming.NewStreamingService(logger, clipReader, cachedNormalizer, playlistGenerator, streamingConfig)

	// Set up session store
	sessionKey, err := dashboard_sessions.GetOrCreateSessionKey()
	if err != nil {
		logger.Error("Failed to get or create session key", err)
		os.Exit(1)
	}
	sessionStore := sessions.NewCookieStore(sessionKey)
	mekStoreFactory := dashboard_sessions.NewMekStoreFactory(sessionStore)

	// Set up Gin engine
	router := gin.Default()

	// Serve static files
	router.Static("/static", "web/static")

	// Set up templates
	router.HTMLRender = createTemplateRenderer()

	// Set up handlers
	authHandler := handlers.NewAuthHandler(logger, mekService, mekStoreFactory)
	clientHandler := handlers.NewClientHandler(logger, clientService, storageManager, mekStoreFactory)
	clipHandler := handlers.NewClipHandler(logger, clipReader, clipDeleter, clientService, mekStoreFactory)
	streamHandler := handlers.NewStreamHandler(logger, streamingService, clientService, mekStoreFactory)

	// Set up middleware
	authMiddleware := middleware.NewAuthMiddleware(logger, mekService, mekStoreFactory)

	// Public routes (authentication)
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/login", authHandler.ShowLogin)
		authGroup.POST("/login", authHandler.Login)
		authGroup.GET("/setup", authHandler.ShowSetup)
		authGroup.POST("/setup", authHandler.Setup)
		authGroup.GET("/logout", authHandler.Logout)
	}

	// Authenticated routes
	authedGroup := router.Group("/")
	authedGroup.Use(authMiddleware.RequireAuth)
	{
		authedGroup.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/clients")
		})

		clientGroup := authedGroup.Group("/clients")
		{
			clientGroup.GET("", clientHandler.ListClients)
			clientGroup.GET("/new", clientHandler.ShowNewClientForm)
			clientGroup.POST("/new", clientHandler.CreateClient)
			clientGroup.POST("/:id/settings", clientHandler.UpdateClientSettings)
			clientGroup.POST("/:id/delete", clientHandler.DeleteClient)
		}

		clipGroup := authedGroup.Group("/clips")
		{
			clipGroup.GET("", clipHandler.ListClips)
			clipGroup.GET("/:id", clipHandler.ViewClip)
			clipGroup.GET("/:id/thumbnail", clipHandler.GetThumbnail)
			clipGroup.GET("/:id/video", clipHandler.GetVideo)
			clipGroup.POST("/delete", clipHandler.DeleteClips)
		}

		streamGroup := authedGroup.Group("/stream")
		{
			streamGroup.GET("", streamHandler.ShowStreamSelection)
			streamGroup.GET("/:clientId", streamHandler.ShowStream)
			streamGroup.GET("/:clientId/playlist.m3u8", streamHandler.GetPlaylist)
			streamGroup.GET("/:clientId/segments/:clipId", streamHandler.GetSegment)
		}
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.WebAddr, cfg.WebPort)
	logger.Info("Starting server on " + addr)
	if err := router.Run(addr); err != nil {
		logger.Error("Failed to start server", err)
	}
}

func createTemplateRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"printf": func(format string, args ...interface{}) string {
			return fmt.Sprintf(format, args...)
		},
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []byte:
				return len(s)
			case string:
				return len(s)
			default:
				return 0
			}
		},
		"formatFileSize": func(bytes int64) string {
			if bytes == 0 {
				return "0 MB"
			}
			mb := float64(bytes) / 1048576.0
			return fmt.Sprintf("%.2f MB", mb)
		},
		"formatBytes": func(bytes int64) string {
			if bytes == 0 {
				return "0 B"
			}
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"toLocal": func(t time.Time) time.Time {
			return t.Local()
		},
		"formatDuration": func(d time.Duration) string {
			// Round to the nearest second to remove milliseconds
			seconds := int(d.Round(time.Second).Seconds())

			if seconds < 60 {
				return fmt.Sprintf("%ds", seconds)
			}

			minutes := seconds / 60
			remainingSeconds := seconds % 60

			if remainingSeconds == 0 {
				return fmt.Sprintf("%dm", minutes)
			}

			return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
		},
		"formatStoragePercent": func(percent float64) string {
			return fmt.Sprintf("%.1f%%", percent)
		},
		"getStorageColorClass": func(percent float64, unlimited bool) string {
			if unlimited {
				return "storage-unlimited"
			}
			if percent >= 90 {
				return "storage-critical"
			} else if percent >= 75 {
				return "storage-warning"
			} else if percent >= 50 {
				return "storage-caution"
			}
			return "storage-ok"
		},
		"sub": func(a, b int64) int64 {
			return a - b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
	}

	r.AddFromFilesFuncs("layout", funcMap, "web/templates/layout.html")
	r.AddFromFilesFuncs("login", funcMap, "web/templates/layout.html", "web/templates/login.html")
	r.AddFromFilesFuncs("setup", funcMap, "web/templates/layout.html", "web/templates/setup.html")
	r.AddFromFilesFuncs("clients", funcMap, "web/templates/layout.html", "web/templates/clients.html")
	r.AddFromFilesFuncs("new-client", funcMap, "web/templates/layout.html", "web/templates/new-client.html")
	r.AddFromFilesFuncs("clips", funcMap, "web/templates/layout.html", "web/templates/clips.html")
	r.AddFromFilesFuncs("clip-detail", funcMap, "web/templates/layout.html", "web/templates/clip-detail.html")
	r.AddFromFilesFuncs("stream-selection", funcMap, "web/templates/layout.html", "web/templates/stream-selection.html")
	r.AddFromFilesFuncs("stream", funcMap, "web/templates/layout.html", "web/templates/stream.html")
	r.AddFromFilesFuncs("error", funcMap, "web/templates/layout.html", "web/templates/error.html")
	return r
}
