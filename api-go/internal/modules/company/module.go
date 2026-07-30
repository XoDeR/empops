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

			sub.Get("/dashboard/me", m.handler.DashboardMe)
			sub.Get("/dashboard/team", m.handler.DashboardTeam)
			sub.Get("/dashboard/manager", m.handler.DashboardManager)
			sub.Get("/dashboard/hr", m.handler.DashboardHR)
			sub.Get("/dashboard/accountant", m.handler.DashboardAccountant)

			sub.With(companyauth.RequirePermission("adminland.access")).Get("/audit-logs", m.handler.ListAuditLogs)

			// Company news: any member may read; create/update/delete are permission-gated.
			sub.Get("/news", m.handler.ListCompanyNews)
			sub.With(companyauth.RequirePermission("news.create")).Post("/news", m.handler.CreateCompanyNews)
			sub.Get("/news/{newsId}", m.handler.ShowCompanyNews)
			sub.With(companyauth.RequirePermission("news.update")).Patch("/news/{newsId}", m.handler.UpdateCompanyNews)
			sub.With(companyauth.RequirePermission("news.delete")).Delete("/news/{newsId}", m.handler.DeleteCompanyNews)

			// Q&A: reads open to any member; writes permission-gated (answers
			// are self-or-permission, checked in the handler).
			sub.Get("/questions", m.handler.ListQuestions)
			sub.Get("/questions/active", m.handler.ActiveQuestion)
			sub.Get("/questions/{questionId}", m.handler.ShowQuestion)
			sub.With(companyauth.RequirePermission("questions.create")).Post("/questions", m.handler.CreateQuestion)
			sub.With(companyauth.RequirePermission("questions.update")).Patch("/questions/{questionId}", m.handler.UpdateQuestion)
			sub.With(companyauth.RequirePermission("questions.delete")).Delete("/questions/{questionId}", m.handler.DeleteQuestion)
			sub.With(companyauth.RequirePermission("questions.manage")).Put("/questions/{questionId}/activate", m.handler.ActivateQuestion)
			sub.With(companyauth.RequirePermission("questions.manage")).Put("/questions/{questionId}/deactivate", m.handler.DeactivateQuestion)
			sub.Post("/questions/{questionId}/answers", m.handler.CreateAnswer)
			sub.Patch("/questions/{questionId}/answers/{answerId}", m.handler.UpdateAnswer)
			sub.Delete("/questions/{questionId}/answers/{answerId}", m.handler.DeleteAnswer)
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
