package company

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	cohttp "github.com/XoDeR/empops/api-go/internal/modules/company/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for the company vertical slice.
type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *cohttp.Handler
}

// New creates an uninitialized company Module.
func New() *Module {
	return &Module{}
}

// Name returns the module identifier used in config/modules.yaml.
func (m *Module) Name() string {
	return "company"
}

// Dependencies returns modules that must initialize before this one.
func (m *Module) Dependencies() []string {
	return nil
}

// Initialize wires handlers using shared Core services.
func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.handler = cohttp.NewHandler(core.DB)
	return nil
}

// RegisterRoutes mounts company and invitation routes under /api/v1.
func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(auth chi.Router) {
		auth.Use(requireAuth)

		auth.Get("/companies", m.handler.ListCompanies)
		auth.Post("/companies", m.handler.CreateCompany)
		auth.Post("/companies/join", m.handler.JoinCompany)
		auth.Post("/invitations/{link}/accept", m.handler.AcceptInvitation)

		auth.Route("/companies/{companyId}", func(sub chi.Router) {
			sub.Use(requireMember)
			sub.Get("/", m.handler.ShowCompany)
			sub.With(companyauth.RequirePermission("company.update")).Patch("/", m.handler.UpdateCompany)
		})
	})
}

// Start begins background work. No-op for company module.
func (m *Module) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down background work. No-op for company module.
func (m *Module) Stop(ctx context.Context) error {
	return nil
}

var _ module.IModule = (*Module)(nil)
