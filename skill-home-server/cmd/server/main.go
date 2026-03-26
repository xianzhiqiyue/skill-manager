package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/skill-home/server/internal/api/handlers"
	"github.com/skill-home/server/internal/api/middleware"
	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/storage"
	"github.com/skill-home/server/internal/webui"
	"github.com/skill-home/server/pkg/validator"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Get()

	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := storage.NewDatabase(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	if err := storage.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to migrate database: %v\n", err)
		os.Exit(1)
	}

	objStorage, err := storage.NewObjectStorage(cfg.Storage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to object storage: %v\n", err)
		os.Exit(1)
	}

	scanner := validator.NewScanner()
	router := setupRouter(db, objStorage, scanner)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		fmt.Printf("Server starting on port %d...\n", cfg.Server.Port)
		fmt.Printf("Environment: %s\n", cfg.Server.Mode)
		fmt.Printf("Version: %s (%s)\n", version, commit)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server forced to shutdown: %v\n", err)
	}

	fmt.Println("Server exited")
}

func setupRouter(db *storage.Database, objStorage *storage.ObjectStorage, scanner *validator.Scanner) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.GET("/health", handlers.HealthCheck)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", handlers.Register(db))
		api.POST("/auth/login", handlers.Login(db))

		api.GET("/skills", handlers.ListSkills(db))
		api.GET("/skills/:namespace/:name", middleware.OptionalAuth(db), handlers.GetSkill(db))
		api.GET("/skills/:namespace/:name/versions", middleware.OptionalAuth(db), handlers.ListVersions(db))
		api.GET("/search", handlers.SearchSkills(db))
		api.GET("/download/:namespace/:name/:version", middleware.OptionalAuth(db), middleware.RateLimit(), handlers.DownloadSkill(db, objStorage))

		auth := api.Group("/")
		auth.Use(middleware.Auth(db))
		{
			auth.POST("/skills", handlers.CreateSkill(db, objStorage, scanner))
			auth.PUT("/skills/:namespace/:name", handlers.UpdateSkill(db))
			auth.DELETE("/skills/:namespace/:name", handlers.DeleteSkill(db, objStorage))
			auth.POST("/skills/:namespace/:name/versions", handlers.PublishVersion(db, objStorage, scanner))
			auth.DELETE("/skills/:namespace/:name/versions/:version", handlers.DeleteVersion(db, objStorage))
			auth.POST("/skills/:namespace/:name/rating", handlers.RateSkill(db))

			auth.GET("/user", handlers.GetCurrentUser)
			auth.GET("/user/skills", handlers.GetUserSkills(db))
			auth.GET("/user/audit-logs", handlers.ListAuditLogs(db))
			auth.GET("/user/api-keys", handlers.ListAPIKeys(db))
			auth.POST("/user/api-keys", handlers.CreateAPIKey(db))
			auth.DELETE("/user/api-keys/:id", handlers.RevokeAPIKey(db))
		}
	}

	if distDir := webui.ResolveDistDir(); distDir != "" {
		webui.Register(r, distDir)
	}

	return r
}
