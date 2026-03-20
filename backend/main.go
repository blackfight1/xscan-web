package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"xscan-web/internal/config"
	"xscan-web/internal/database"
	"xscan-web/internal/handler"
	"xscan-web/internal/scanner"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("XScan Web Server starting on port %d", cfg.Port)

	// Initialize database
	database.Init(cfg.DBPath)
	defer database.Close()

	// Initialize scanner
	sc := scanner.New(scanner.Config{
		XscanPath:     cfg.XscanPath,
		ToolsDir:      cfg.ToolsDir,
		ResultsDir:    cfg.ResultsDir,
		MaxConcurrent: cfg.MaxConcurrent,
	})
	defer sc.Stop()

	// Initialize handler
	h := handler.New(sc)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Auth middleware
	authMiddleware := func(c *gin.Context) {
		if cfg.AuthToken == "" {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		if token != cfg.AuthToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	// API routes
	api := r.Group("/api", authMiddleware)
	{
		api.POST("/tasks", h.CreateTask)
		api.GET("/tasks", h.GetTasks)
		api.GET("/tasks/:id", h.GetTask)
		api.DELETE("/tasks/:id", h.DeleteTask)
		api.GET("/tasks/:id/report", h.GetReport)
	}

	// Health check (no auth)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Serve frontend static files
	r.StaticFS("/assets", http.Dir("./static/assets"))
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.NoRoute(func(c *gin.Context) {
		// For SPA routing, serve index.html for non-API routes
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.File("./static/index.html")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server running at http://0.0.0.0%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
