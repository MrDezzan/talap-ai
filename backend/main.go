package main

import (
	"log"
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
	"talkd-backend/middleware"
	"talkd-backend/models"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	db := config.InitDB()

	if err := db.AutoMigrate(&models.User{}, &models.Grant{}, &models.Profession{}, &models.Portfolio{}, &models.ChatThread{}, &models.ChatMessage{}); err != nil {
		log.Printf("AutoMigrate error: %v", err)
	}
	config.SeedDB(db)

	// Must start after AutoMigrate/SeedDB so the Grant table (and its
	// status/source_url/fetched_at columns) definitely exists before the
	// importer's first immediate run tries to query/insert into it.
	importer.StartScheduled(db, 6*time.Hour)

	r := gin.Default()

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

	// gin.Default() below already installs Logger + Recovery middleware,
	// so we only need to add CORS on top of it.
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

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
