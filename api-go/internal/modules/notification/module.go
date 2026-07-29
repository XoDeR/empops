// Package notification implements the vertical slice for in-app employee
// notifications (list + mark-as-read). Rows are inserted by other modules
// via pkg/notify (e.g. the team module when attaching employees to a ship).
package notification

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	notificationhttp "github.com/XoDeR/empops/api-go/internal/modules/notification/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for notifications.
type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *notificationhttp.Handler
}

// New creates an uninitialized notification Module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "notification" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee"}
}

func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.handler = notificationhttp.NewHandler(core.DB)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)

		sub.Get("/companies/{companyId}/notifications", m.handler.ListNotifications)
		sub.Post("/companies/{companyId}/notifications/read", m.handler.MarkRead)
	})
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

var _ module.IModule = (*Module)(nil)
