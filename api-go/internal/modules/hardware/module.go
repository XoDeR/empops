// Package hardware implements hardware assets and software license inventory (Step 9).
package hardware

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/internal/modules/finance/adapter/frankfurter"
	hardwarehttp "github.com/XoDeR/empops/api-go/internal/modules/hardware/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *hardwarehttp.Handler
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "hardware" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee", "finance", "media"}
}

func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool, m.jwt = core.DB, core.JWT
	m.handler = hardwarehttp.NewHandler(core.DB, frankfurter.New())
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(sub chi.Router) {
		sub.Use(httpauth.RequireAuth(m.jwt))
		sub.Use(companyauth.RequireMember(m.pool))
		base := "/companies/{companyId}"

		sub.With(companyauth.RequirePermission("hardware.view")).Get(base+"/hardware", m.handler.ListHardware)
		sub.With(companyauth.RequirePermission("hardware.manage")).Post(base+"/hardware", m.handler.CreateHardware)
		sub.With(companyauth.RequirePermission("hardware.view")).Get(base+"/hardware/{hardwareId}", m.handler.ShowHardware)
		sub.With(companyauth.RequirePermission("hardware.manage")).Patch(base+"/hardware/{hardwareId}", m.handler.UpdateHardware)
		sub.With(companyauth.RequirePermission("hardware.manage")).Delete(base+"/hardware/{hardwareId}", m.handler.DestroyHardware)
		sub.With(companyauth.RequirePermission("hardware.manage")).Post(base+"/hardware/{hardwareId}/lend", m.handler.LendHardware)
		sub.With(companyauth.RequirePermission("hardware.manage")).Post(base+"/hardware/{hardwareId}/regain", m.handler.RegainHardware)
		sub.Get(base+"/employees/{employeeId}/hardware", m.handler.EmployeeHardware)

		sub.With(companyauth.RequirePermission("software.view")).Get(base+"/softwares", m.handler.ListSoftwares)
		sub.With(companyauth.RequirePermission("software.manage")).Post(base+"/softwares", m.handler.CreateSoftware)
		sub.With(companyauth.RequirePermission("software.view")).Get(base+"/softwares/{softwareId}", m.handler.ShowSoftware)
		sub.With(companyauth.RequirePermission("software.manage")).Patch(base+"/softwares/{softwareId}", m.handler.UpdateSoftware)
		sub.With(companyauth.RequirePermission("software.manage")).Delete(base+"/softwares/{softwareId}", m.handler.DestroySoftware)
		sub.With(companyauth.RequirePermission("software.manage")).Post(base+"/softwares/{softwareId}/seats", m.handler.GiveSeat)
		sub.With(companyauth.RequirePermission("software.manage")).Post(base+"/softwares/{softwareId}/seats/all", m.handler.GiveSeatsToAll)
		sub.With(companyauth.RequirePermission("software.manage")).Delete(base+"/softwares/{softwareId}/seats/{employeeId}", m.handler.RevokeSeat)
		sub.With(companyauth.RequirePermission("software.view")).Get(base+"/softwares/{softwareId}/employees-without", m.handler.EmployeesWithout)
		sub.With(companyauth.RequirePermission("software.manage")).Post(base+"/softwares/{softwareId}/files", m.handler.AttachSoftwareFile)
		sub.With(companyauth.RequirePermission("software.manage")).Delete(base+"/softwares/{softwareId}/files/{mediaId}", m.handler.DetachSoftwareFile)
		sub.Get(base+"/employees/{employeeId}/softwares", m.handler.EmployeeSoftwares)
	})
}

func (m *Module) Start(context.Context) error { return nil }
func (m *Module) Stop(context.Context) error  { return nil }

var _ module.IModule = (*Module)(nil)
