package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/encryption"
	"github.com/yeti47/cryospy/server/core/videos"
	"github.com/yeti47/cryospy/server/dashboard/config"
	dashboard_sessions "github.com/yeti47/cryospy/server/dashboard/sessions"
	"github.com/yeti47/cryospy/server/dashboard/web/handlers"
	"github.com/yeti47/cryospy/server/dashboard/web/middleware"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// try to save the config in case it was not found
	if err := cfg.SaveConfig("config.json"); err != nil {
		log.Printf("Failed to save configuration: %v", err)
	}

	// Set up logger
	logger := logging.CreateLogger(logging.LogLevel(cfg.LogLevel), cfg.LogPath, "dashboard")

	// Set up database connection
	dbConn, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to open database", err)
		os.Exit(1)
	}
	defer dbConn.Close()

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
	clientHandler := handlers.NewClientHandler(logger, clientService, mekStoreFactory)
	clipHandler := handlers.NewClipHandler(logger, clipReader, mekStoreFactory)

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
			clipGroup.GET("/:id/thumbnail", clipHandler.GetThumbnail)
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
	}

	r.AddFromFilesFuncs("layout", funcMap, "web/templates/layout.html")
	r.AddFromFilesFuncs("login", funcMap, "web/templates/layout.html", "web/templates/login.html")
	r.AddFromFilesFuncs("setup", funcMap, "web/templates/layout.html", "web/templates/setup.html")
	r.AddFromFilesFuncs("clients", funcMap, "web/templates/layout.html", "web/templates/clients.html")
	r.AddFromFilesFuncs("new-client", funcMap, "web/templates/layout.html", "web/templates/new-client.html")
	r.AddFromFilesFuncs("clips", funcMap, "web/templates/layout.html", "web/templates/clips.html")
	return r
}
