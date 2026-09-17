package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"talkd-backend/config"
	"talkd-backend/handlers"
	"talkd-backend/importer"
	"talkd-backend/logger"
	"talkd-backend/middleware"
	"talkd-backend/migrations"
	"talkd-backend/models"
	"talkd-backend/monitoring"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, reading from environment")
	}

	db := config.InitDB()

	if err := db.AutoMigrate(&models.User{}, &models.Grant{}, &models.Profession{}, &models.Portfolio{}, &models.ChatThread{}, &models.ChatMessage{}); err != nil {
		logger.Error("AutoMigrate error", "error", err)
	}
	config.SeedDB(db)

	// Run versioned migrations after AutoMigrate/SeedDB, so the DB is
	// already in its normal state. This only affects future explicit
	// migrations (see backend/migrations), not current functionality, so
	// we log and continue rather than crashing the app on failure.
	if err := migrations.Migrate(db); err != nil {
		logger.Error("migrations: failed to apply", "error", err)
		monitoring.Alert("migrations_failed", map[string]any{"error": err.Error()})
	}

	// Must start after AutoMigrate/SeedDB so the Grant table (and its
	// status/source_url/fetched_at columns) definitely exists before the
	// importer's first immediate run tries to query/insert into it.
	importer.StartScheduled(db, 6*time.Hour)

	r := gin.Default()
	// gin.Default() installs its own panic recovery; replace it with a
	// custom one so HTTP-request-level panics also fire an alert, in
	// addition to responding with a generic 500.
	r.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		logger.Error("http: recovered from panic", "panic", fmt.Sprint(recovered), "path", c.Request.URL.Path)
		monitoring.Alert("panic", map[string]any{"error": fmt.Sprint(recovered), "path": c.Request.URL.Path})
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}))

	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGINS")
	corsConfig := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}

	if allowedOrigin == "" || allowedOrigin == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(allowedOrigin, ",")
	}

	// gin.Default() installs its own Logger middleware; recovery was
	// replaced above with our custom handler, so we only need to add
	// CORS on top of it here.
	r.Use(cors.New(corsConfig))

	// Lightweight healthcheck for Railway/load balancers — no DB round trip
	// needed, just confirms the process is up and serving.
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/register", middleware.RateLimit(5, 5), handlers.Register(db))
		api.POST("/login", middleware.RateLimit(5, 5), handlers.Login(db))
		api.GET("/auth/google/login", handlers.GoogleLogin)
		api.GET("/auth/google/callback", handlers.GoogleCallback(db))

		api.GET("/grants", handlers.GetGrants(db))
		api.GET("/professions", handlers.GetProfessions(db))
		api.GET("/search", handlers.GlobalSearch(db))

		auth := api.Group("/")
		auth.Use(middleware.AuthRequired())
		{
			auth.GET("/me", handlers.GetMe(db))
			auth.PUT("/me", handlers.UpdateMe(db))
			auth.GET("/dashboard", handlers.GetDashboardData(db))
			auth.GET("/portfolio", handlers.GetPortfolio(db))
			auth.PUT("/portfolio", handlers.UpdatePortfolio(db))
			auth.POST("/ai/analyze", middleware.RateLimit(12, 4), handlers.AnalyzeProfile(db))
			auth.POST("/ai/chat", middleware.RateLimit(12, 4), handlers.AIChat(db))
			auth.PUT("/portfolio/roadmap", handlers.UpdatePortfolioRoadmap(db))
			auth.GET("/chats", handlers.GetChatThreads(db))
			auth.GET("/chats/:id/messages", handlers.GetChatMessages(db))

			admin := auth.Group("/admin")
			admin.Use(middleware.AdminRequired(db))
			{
				admin.GET("/stats", handlers.GetAdminStats(db))
				admin.POST("/grants", handlers.CreateGrant(db))
				admin.PUT("/grants/:id", handlers.UpdateGrant(db))
				admin.DELETE("/grants/:id", handlers.DeleteGrant(db))
				admin.GET("/grants/pending", handlers.GetPendingGrants(db))
				admin.POST("/grants/:id/approve", handlers.ApproveGrant(db))
				admin.POST("/grants/:id/reject", handlers.RejectGrant(db))
				admin.POST("/professions", handlers.CreateProfession(db))
				admin.PUT("/professions/:id", handlers.UpdateProfession(db))
				admin.DELETE("/professions/:id", handlers.DeleteProfession(db))
			}
		}
	}

	// Serve Static Files for Production
	r.Static("/assets", "./dist/assets")
	r.StaticFile("/logo.svg", "./dist/logo.svg")
	r.StaticFile("/favicon.ico", "./dist/favicon.ico")
	
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		c.File("./dist/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting on port", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("server exited", "error", err)
	}
}
