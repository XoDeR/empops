// Command api is the Core composition root: config, Postgres, JWT auth,
// Chi router, and enabled vertical modules.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/file-uploads-go/backend/pkg/upload"
	"github.com/file-uploads-go/backend/pkg/upload/storage"

	httpadapter "github.com/XoDeR/empops/api-go/internal/adapter/http"
	"github.com/XoDeR/empops/api-go/internal/adapter/persistence"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
	"github.com/XoDeR/empops/api-go/internal/usecase"
	"github.com/XoDeR/empops/api-go/pkg/bus"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/logger"
	"github.com/XoDeR/empops/api-go/pkg/module"

	_ "github.com/XoDeR/empops/api-go/internal/modules/company"
	_ "github.com/XoDeR/empops/api-go/internal/modules/employee"
	_ "github.com/XoDeR/empops/api-go/internal/modules/finance"
	_ "github.com/XoDeR/empops/api-go/internal/modules/grow"
	_ "github.com/XoDeR/empops/api-go/internal/modules/media"
	_ "github.com/XoDeR/empops/api-go/internal/modules/notification"
	_ "github.com/XoDeR/empops/api-go/internal/modules/place"
	_ "github.com/XoDeR/empops/api-go/internal/modules/project"
	_ "github.com/XoDeR/empops/api-go/internal/modules/recruit"
	_ "github.com/XoDeR/empops/api-go/internal/modules/team"
	_ "github.com/XoDeR/empops/api-go/internal/modules/time"

	mediahttp "github.com/XoDeR/empops/api-go/internal/modules/media/adapter/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	modulesPath := envOrDefault("EMPOPS_MODULES_CONFIG", "config/modules.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	modulesCfg, err := config.LoadModules(modulesPath)
	if err != nil {
		return fmt.Errorf("load modules config: %w", err)
	}

	if cfg.DB.DSN == "" {
		return fmt.Errorf("database DSN is required (set EMPOPS_DB_DSN or db.dsn in %s)", configPath)
	}

	log := logger.New(cfg.Log.Level)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.Connect(ctx, database.Config{
		DSN:            cfg.DB.DSN,
		ConnectTimeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	jwtManager := jwt.NewManager(jwt.Config{
		Secret:          cfg.JWT.Secret,
		Issuer:          cfg.JWT.Issuer,
		Audience:        cfg.JWT.Audience,
		AccessTokenTTL:  cfg.JWT.AccessTTL(),
		RefreshTokenTTL: cfg.JWT.RefreshTTL(),
	})

	userRepo := persistence.NewPostgresUserRepository(pool)
	refreshRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	authUseCase := usecase.NewAuthUseCase(userRepo, refreshRepo, jwtManager)

	eventBus := bus.NewMemoryBus()
	core := &module.Core{
		Logger: log,
		JWT:    jwtManager,
		Bus:    eventBus,
		DB:     pool,
	}

	uploadDir := envOrDefault("EMPOPS_UPLOAD_DIR", "./uploads")
	uploadMaxSizeBytes := int64(100 * 1024 * 1024) // 100MB
	if raw := os.Getenv("EMPOPS_UPLOAD_MAX_SIZE_BYTES"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			uploadMaxSizeBytes = v
		}
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("upload: create upload dir: %w", err)
	}

	uploadStore := storage.NewLocal(uploadDir)
	uploadSvc, err := upload.NewService(upload.Options{
		Config: upload.Config{
			UploadDir: uploadDir,
			MaxSize:   uploadMaxSizeBytes,
		},
		Storage: uploadStore,
	})
	if err != nil {
		return fmt.Errorf("upload: create upload service: %w", err)
	}

	mediaHandler := mediahttp.NewHandler(pool, uploadDir)

	uploadRoutes := func(up chi.Router) {
		up.Use(uploadSvc.RateLimiter().Middleware)

		// Stream multipart endpoint (multi-file) - parity with file-uploads-go.
		up.Post("/stream", uploadSvc.HandleStream)

		// Chunked resumable endpoint (single-file upload session per upload_id).
		cm := uploadSvc.ChunkedManager()
		up.Post("/init", cm.InitiateUpload)
		up.Post("/chunk", cm.UploadChunk)
		up.Post("/complete", mediaHandler.WrapComplete(cm))
		up.Get("/status", cm.GetUploadStatus)

		// Optional: SSE progress (primarily used by stream uploader).
		up.Get("/progress", uploadSvc.ProgressTracker().SSEHandler)
	}

	initialized, err := module.DefaultRegistry.InitializeAll(ctx, core, modulesCfg.Enabled)
	if err != nil {
		return fmt.Errorf("initialize modules: %w", err)
	}

	if err := module.StartAll(ctx, initialized); err != nil {
		return fmt.Errorf("start modules: %w", err)
	}
	defer module.StopAll(ctx, initialized)

	handler := httpadapter.NewRouter(httpadapter.RouterConfig{
		AuthUseCase:    authUseCase,
		JWTManager:     jwtManager,
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		UploadRoutes:   uploadRoutes,
		RegisterModules: func(r chi.Router) {
			module.RegisterRoutes(r, initialized)
		},
	})

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	case sig := <-quit:
		log.Info("shutting down", "signal", sig.String())
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
