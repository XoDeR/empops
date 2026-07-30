// Package time implements timesheets and work-from-home tracking.
package time

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	timehttp "github.com/XoDeR/empops/api-go/internal/modules/time/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct {
	pool *pgxpool.Pool
	jwt *jwt.Manager
	handler *timehttp.Handler
}

func New() *Module { return &Module{} }
func (m *Module) Name() string { return "time" }
func (m *Module) Dependencies() []string { return []string{"company", "employee"} }
func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool, m.jwt = core.DB, core.JWT
	m.handler = timehttp.NewHandler(core.DB)
	return nil
}
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(sub chi.Router) {
		sub.Use(httpauth.RequireAuth(m.jwt))
		sub.Use(companyauth.RequireMember(m.pool))
		base := "/companies/{companyId}"
		sub.With(companyauth.RequirePermission("timesheets.view")).Get(base+"/timesheets", m.handler.Timesheet)
		sub.With(companyauth.RequirePermission("timesheets.view")).Post(base+"/timesheets", m.handler.Timesheet)
		sub.With(companyauth.RequirePermission("timesheets.approve")).Get(base+"/timesheets/pending", m.handler.Pending)
		sub.With(companyauth.RequirePermission("timesheets.view")).Get(base+"/timesheets/{timesheetId}", m.handler.Show)
		sub.With(companyauth.RequirePermission("timesheets.view")).Post(base+"/timesheets/{timesheetId}/entries", m.handler.UpsertEntry)
		sub.With(companyauth.RequirePermission("timesheets.view")).Delete(base+"/timesheets/{timesheetId}/entries/{entryId}", m.handler.DeleteEntry)
		sub.With(companyauth.RequirePermission("timesheets.view")).Post(base+"/timesheets/{timesheetId}/submit", m.handler.Submit)
		sub.With(companyauth.RequirePermission("timesheets.approve")).Post(base+"/timesheets/{timesheetId}/approve", m.handler.Approve)
		sub.With(companyauth.RequirePermission("timesheets.approve")).Post(base+"/timesheets/{timesheetId}/reject", m.handler.Reject)
		sub.Put(base+"/employees/{employeeId}/work-from-home", m.handler.SetWorkFromHome)
		sub.Get(base+"/work-from-home", m.handler.WorkFromHomeSetting)
		sub.Patch(base+"/work-from-home", m.handler.UpdateWorkFromHomeSetting)
	})
}
func (m *Module) Start(context.Context) error { return nil }
func (m *Module) Stop(context.Context) error { return nil }

var _ module.IModule = (*Module)(nil)
