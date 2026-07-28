package httpadapter

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/XoDeR/empops/api-go/internal/usecase"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
)

// RouterConfig holds everything NewRouter needs to build the Core router.
type RouterConfig struct {
	AuthUseCase    *usecase.AuthUseCase
	JWTManager     *jwt.Manager
	AllowedOrigins []string
	// RegisterModules mounts every enabled module's routes under /api/v1
	// once Core routes are in place (blank-imported in cmd/api).
	RegisterModules func(r chi.Router)
}

// NewRouter builds the Chi router: middleware, CORS, health/version, Core
// auth routes and (via RegisterModules) every enabled module's routes.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authHandler := NewAuthHandler(cfg.AuthUseCase)
	requireAuth := httpauth.RequireAuth(cfg.JWTManager)

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", HealthHandler)
		v1.Get("/version", VersionHandler)

		v1.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", authHandler.Register)
			auth.Post("/login", authHandler.Login)
			auth.Post("/refresh", authHandler.Refresh)
			auth.Post("/logout", authHandler.Logout)

			auth.Group(func(protected chi.Router) {
				protected.Use(requireAuth)
				protected.Get("/me", authHandler.Me)
			})
		})

		if cfg.RegisterModules != nil {
			cfg.RegisterModules(v1)
		}
	})

	return r
}
