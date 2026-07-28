package team

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	teamhttp "github.com/XoDeR/empops/api-go/internal/modules/team/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for the team vertical slice.
type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *teamhttp.Handler
}

// New creates an uninitialized team Module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "team" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee"}
}

func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.handler = teamhttp.NewHandler(core.DB)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)

		sub.With(companyauth.RequirePermission("teams.view")).Get("/companies/{companyId}/teams", m.handler.ListTeams)
		sub.With(companyauth.RequirePermission("teams.create")).Post("/companies/{companyId}/teams", m.handler.CreateTeam)
		sub.With(companyauth.RequirePermission("teams.view")).Get("/companies/{companyId}/teams/{teamId}", m.handler.ShowTeam)
		sub.With(companyauth.RequirePermission("teams.update")).Patch("/companies/{companyId}/teams/{teamId}", m.handler.UpdateTeam)
		sub.With(companyauth.RequirePermission("teams.delete")).Delete("/companies/{companyId}/teams/{teamId}", m.handler.DeleteTeam)

		sub.With(companyauth.RequirePermission("teams.manage_members")).Post("/companies/{companyId}/teams/{teamId}/members/{employeeId}", m.handler.AddMember)
		sub.With(companyauth.RequirePermission("teams.manage_members")).Delete("/companies/{companyId}/teams/{teamId}/members/{employeeId}", m.handler.RemoveMember)
		sub.With(companyauth.RequirePermission("teams.manage_members")).Put("/companies/{companyId}/teams/{teamId}/lead", m.handler.SetLead)
	})
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

var _ module.IModule = (*Module)(nil)
