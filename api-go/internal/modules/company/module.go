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

			sub.With(companyauth.RequirePermission("flows.manage")).Get("/flows", m.handler.ListFlows)
			sub.With(companyauth.RequirePermission("flows.manage")).Post("/flows", m.handler.CreateFlow)
			sub.With(companyauth.RequirePermission("flows.manage")).Get("/flows/{flowId}", m.handler.ShowFlow)
			sub.With(companyauth.RequirePermission("flows.manage")).Patch("/flows/{flowId}", m.handler.UpdateFlow)
			sub.With(companyauth.RequirePermission("flows.manage")).Delete("/flows/{flowId}", m.handler.DeleteFlow)
			sub.With(companyauth.RequirePermission("flows.manage")).Post("/flows/{flowId}/steps", m.handler.CreateFlowStep)
			sub.With(companyauth.RequirePermission("flows.manage")).Patch("/flows/{flowId}/steps/{stepId}", m.handler.UpdateFlowStep)
			sub.With(companyauth.RequirePermission("flows.manage")).Delete("/flows/{flowId}/steps/{stepId}", m.handler.DeleteFlowStep)
			sub.With(companyauth.RequirePermission("flows.manage")).Post("/flows/{flowId}/steps/{stepId}/actions", m.handler.CreateFlowAction)
			sub.With(companyauth.RequirePermission("flows.manage")).Patch("/flows/{flowId}/steps/{stepId}/actions/{actionId}", m.handler.UpdateFlowAction)
			sub.With(companyauth.RequirePermission("flows.manage")).Delete("/flows/{flowId}/steps/{stepId}/actions/{actionId}", m.handler.DeleteFlowAction)

			sub.With(companyauth.RequirePermission("wiki.view")).Get("/wikis", m.handler.ListWikis)
			sub.With(companyauth.RequirePermission("wiki.create")).Post("/wikis", m.handler.CreateWiki)
			sub.With(companyauth.RequirePermission("wiki.view")).Get("/wikis/{wikiId}", m.handler.ShowWiki)
			sub.With(companyauth.RequirePermission("wiki.update")).Patch("/wikis/{wikiId}", m.handler.UpdateWiki)
			sub.With(companyauth.RequirePermission("wiki.delete")).Delete("/wikis/{wikiId}", m.handler.DeleteWiki)
			sub.With(companyauth.RequirePermission("wiki.create")).Post("/wikis/{wikiId}/pages", m.handler.CreateWikiPage)
			sub.With(companyauth.RequirePermission("wiki.view")).Get("/wikis/{wikiId}/pages/{pageId}", m.handler.ShowWikiPage)
			sub.With(companyauth.RequirePermission("wiki.update")).Patch("/wikis/{wikiId}/pages/{pageId}", m.handler.UpdateWikiPage)
			sub.With(companyauth.RequirePermission("wiki.delete")).Delete("/wikis/{wikiId}/pages/{pageId}", m.handler.DeleteWikiPage)
			sub.With(companyauth.RequirePermission("wiki.view")).Get("/wikis/{wikiId}/pages/{pageId}/revisions", m.handler.ListWikiPageRevisions)

			sub.With(companyauth.RequirePermission("ama.view")).Get("/ama-sessions", m.handler.ListAMASessions)
			sub.With(companyauth.RequirePermission("ama.manage")).Post("/ama-sessions", m.handler.CreateAMASession)
			sub.With(companyauth.RequirePermission("ama.view")).Get("/ama-sessions/{sessionId}", m.handler.ShowAMASession)
			sub.With(companyauth.RequirePermission("ama.manage")).Patch("/ama-sessions/{sessionId}", m.handler.UpdateAMASession)
			sub.With(companyauth.RequirePermission("ama.manage")).Delete("/ama-sessions/{sessionId}", m.handler.DeleteAMASession)
			sub.With(companyauth.RequirePermission("ama.view")).Get("/ama-sessions/{sessionId}/questions", m.handler.ListAMAQuestions)
			sub.With(companyauth.RequirePermission("ama.view")).Post("/ama-sessions/{sessionId}/questions", m.handler.CreateAMAQuestion)
			sub.With(companyauth.RequirePermission("ama.manage")).Patch("/ama-sessions/{sessionId}/questions/{questionId}", m.handler.UpdateAMAQuestion)
			sub.With(companyauth.RequirePermission("ama.manage")).Delete("/ama-sessions/{sessionId}/questions/{questionId}", m.handler.DeleteAMAQuestion)
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
