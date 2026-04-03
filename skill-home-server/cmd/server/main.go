package main

import (
	"context"
	"fmt"
	"io"
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
	handled, err := handleVersionCommand(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to print version: %v\n", err)
		os.Exit(1)
	}
	if handled {
		return
	}

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
	if err := handlers.EnsureBootstrapSuperAdmin(db, cfg.Auth.BootstrapSuperAdmin); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bootstrap super admin: %v\n", err)
		os.Exit(1)
	}

	objStorage, err := storage.NewObjectStorage(cfg.Storage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to object storage: %v\n", err)
		os.Exit(1)
	}

	scanner := validator.NewScanner()
	router := setupRouter(db, objStorage, scanner, cfg.Server.BasePath)

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

func setupRouter(db *storage.Database, objStorage *storage.ObjectStorage, scanner *validator.Scanner, basePath string) *gin.Engine {
	r := gin.New()
	basePath = config.NormalizeBasePath(basePath)

	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	r.GET(config.JoinBasePath(basePath, "/health"), handlers.HealthCheck(version))

	api := r.Group(config.JoinBasePath(basePath, "/api/v1"))
	{
		api.GET("/catalog/version", handlers.GetCatalogVersion(db))
		api.POST("/auth/register", handlers.Register(db))
		api.POST("/auth/login", handlers.Login(db))

		api.GET("/skills", handlers.ListSkills(db, objStorage))
		api.GET("/skills/:namespace/:name", middleware.OptionalAuth(db), handlers.GetSkill(db, objStorage))
		api.GET("/skills/:namespace/:name/versions", middleware.OptionalAuth(db), handlers.ListVersions(db, objStorage))
		api.GET("/search", handlers.SearchSkills(db, objStorage))
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
			auth.GET("/admin/users", handlers.ListUsers(db))
			auth.PUT("/admin/users/:id", handlers.UpdateUserByAdmin(db))
		}
	}

	if distDir := webui.ResolveDistDir(); distDir != "" {
		webui.Register(r, distDir, webui.Options{BasePath: basePath})
	}

	return r
}

func handleVersionCommand(args []string, out io.Writer) (bool, error) {
	if len(args) != 1 {
		return false, nil
	}

	switch args[0] {
	case "--version", "version":
		_, err := fmt.Fprintf(out, "skill-home-server\n  Version:   %s\n  Commit:    %s\n  BuildDate: %s\n", version, commit, buildDate)
		return true, err
	default:
		return false, nil
	}
}
