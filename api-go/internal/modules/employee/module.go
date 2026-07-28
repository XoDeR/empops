package employee

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	emhttp "github.com/XoDeR/empops/api-go/internal/modules/employee/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for the employee vertical slice.
type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *emhttp.Handler
}

// New creates an uninitialized employee Module.
func New() *Module {
	return &Module{}
}

// Name returns the module identifier used in config/modules.yaml.
func (m *Module) Name() string {
	return "employee"
}

// Dependencies returns modules that must initialize before this one.
func (m *Module) Dependencies() []string {
	return []string{"company"}
}

// Initialize wires handlers using shared Core services.
func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.handler = emhttp.NewHandler(core.DB)
	return nil
}

// RegisterRoutes mounts employee routes under /companies/{companyId}/.
func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	// Full paths (not Route("/companies/{companyId}")) — company module already
	// owns that prefix for GET/PATCH company settings.
	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)

		sub.With(companyauth.RequirePermission("employees.view")).Get("/companies/{companyId}/employees", m.handler.ListEmployees)
		sub.With(companyauth.RequirePermission("employees.create")).Post("/companies/{companyId}/employees", m.handler.CreateEmployee)
		sub.With(companyauth.RequirePermission("employees.view")).Get("/companies/{companyId}/employees/{employeeId}", m.handler.ShowEmployee)
		sub.Patch("/companies/{companyId}/employees/{employeeId}", m.handler.UpdateEmployee)
		sub.With(companyauth.RequirePermission("employees.delete")).Delete("/companies/{companyId}/employees/{employeeId}", m.handler.DeleteEmployee)
		sub.With(companyauth.RequirePermission("employees.invite")).Post("/companies/{companyId}/employees/{employeeId}/invite", m.handler.InviteEmployee)

		sub.With(companyauth.RequirePermission("employees.view")).Get("/companies/{companyId}/employees/{employeeId}/managers", m.handler.ListManagers)
		sub.With(companyauth.RequirePermission("employees.view")).Get("/companies/{companyId}/employees/{employeeId}/direct-reports", m.handler.ListDirectReports)
		sub.With(companyauth.RequirePermission("hierarchy.assign")).Post("/companies/{companyId}/employees/{employeeId}/managers", m.handler.AssignManager)
		sub.With(companyauth.RequirePermission("hierarchy.assign")).Delete("/companies/{companyId}/employees/{employeeId}/managers/{managerId}", m.handler.UnassignManager)

		sub.With(companyauth.RequirePermission("positions.view")).Get("/companies/{companyId}/positions", m.handler.ListPositions)
		sub.With(companyauth.RequirePermission("positions.create")).Post("/companies/{companyId}/positions", m.handler.CreatePosition)
		sub.With(companyauth.RequirePermission("positions.update")).Patch("/companies/{companyId}/positions/{positionId}", m.handler.UpdatePosition)
		sub.With(companyauth.RequirePermission("positions.delete")).Delete("/companies/{companyId}/positions/{positionId}", m.handler.DeletePosition)

		sub.With(companyauth.RequirePermission("employee-statuses.view")).Get("/companies/{companyId}/employee-statuses", m.handler.ListEmployeeStatuses)
		sub.With(companyauth.RequirePermission("employee-statuses.create")).Post("/companies/{companyId}/employee-statuses", m.handler.CreateEmployeeStatus)
		sub.With(companyauth.RequirePermission("employee-statuses.update")).Patch("/companies/{companyId}/employee-statuses/{statusId}", m.handler.UpdateEmployeeStatus)
		sub.With(companyauth.RequirePermission("employee-statuses.delete")).Delete("/companies/{companyId}/employee-statuses/{statusId}", m.handler.DeleteEmployeeStatus)
	})
}

// Start begins background work. No-op for employee module.
func (m *Module) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down background work. No-op for employee module.
func (m *Module) Stop(ctx context.Context) error {
	return nil
}

var _ module.IModule = (*Module)(nil)
