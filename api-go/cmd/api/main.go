// Command api is the Core composition root: it loads config, wires JWT and
// the stub auth use case, builds the Chi router, initializes every enabled
// module from the registry, and serves HTTP.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	httpadapter "github.com/XoDeR/empops/api-go/internal/adapter/http"
	"github.com/XoDeR/empops/api-go/internal/adapter/persistence"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/usecase"
	"github.com/XoDeR/empops/api-go/pkg/bus"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/logger"
	"github.com/XoDeR/empops/api-go/pkg/module"

	// Blank-import enabled modules so their init() registers them into
	// module.DefaultRegistry. Add new modules here as they ship.
	_ "github.com/XoDeR/empops/api-go/internal/modules/example"
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

	log := logger.New(cfg.Log.Level)

	jwtManager := jwt.NewManager(jwt.Config{
		Secret:          cfg.JWT.Secret,
		Issuer:          cfg.JWT.Issuer,
		Audience:        cfg.JWT.Audience,
		AccessTokenTTL:  cfg.JWT.AccessTTL(),
		RefreshTokenTTL: cfg.JWT.RefreshTTL(),
	})

	// Step 0 stub: in-memory user repository, no Postgres connection needed.
	userRepo := persistence.NewMemoryUserRepository()
	authUseCase := usecase.NewAuthUseCase(userRepo, jwtManager)

	eventBus := bus.NewMemoryBus()
	core := &module.Core{
		Logger: log,
		JWT:    jwtManager,
		Bus:    eventBus,
		DB:     nil, // Step 0 stub: no database wired up yet
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
