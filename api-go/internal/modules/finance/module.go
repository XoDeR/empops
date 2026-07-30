// Package finance implements expense categories, approvals, and FX conversion.
package finance

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/internal/modules/finance/adapter/frankfurter"
	financehttp "github.com/XoDeR/empops/api-go/internal/modules/finance/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct { pool *pgxpool.Pool; jwt *jwt.Manager; handler *financehttp.Handler }
func New() *Module { return &Module{} }
func (m *Module) Name() string { return "finance" }
func (m *Module) Dependencies() []string { return []string{"company","employee"} }
func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool,m.jwt=core.DB,core.JWT
	m.handler=financehttp.NewHandler(core.DB,frankfurter.New())
	return nil
}
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(sub chi.Router){
		sub.Use(httpauth.RequireAuth(m.jwt));sub.Use(companyauth.RequireMember(m.pool))
		base:="/companies/{companyId}"
		sub.With(companyauth.RequirePermission("expenses.view")).Get(base+"/expense-categories",m.handler.Categories)
		sub.With(companyauth.RequirePermission("expenses.manage_categories")).Post(base+"/expense-categories",m.handler.CreateCategory)
		sub.With(companyauth.RequirePermission("expenses.manage_categories")).Patch(base+"/expense-categories/{categoryId}",m.handler.UpdateCategory)
		sub.With(companyauth.RequirePermission("expenses.manage_categories")).Delete(base+"/expense-categories/{categoryId}",m.handler.DeleteCategory)
		sub.With(companyauth.RequirePermission("expenses.view")).Get(base+"/expenses",m.handler.Expenses)
		sub.With(companyauth.RequirePermission("expenses.view")).Post(base+"/expenses",m.handler.CreateExpense)
		sub.With(companyauth.RequirePermission("expenses.view")).Get(base+"/expenses/pending/manager",m.handler.PendingManager)
		sub.With(companyauth.RequirePermission("expenses.finalize")).Get(base+"/expenses/pending/accounting",m.handler.PendingAccounting)
		sub.With(companyauth.RequirePermission("expenses.view")).Get(base+"/expenses/{expenseId}",m.handler.ShowExpense)
		sub.With(companyauth.RequirePermission("expenses.view")).Delete(base+"/expenses/{expenseId}",m.handler.DeleteExpense)
		sub.With(companyauth.RequirePermission("expenses.view")).Post(base+"/expenses/{expenseId}/manager-approve",m.handler.ManagerApprove)
		sub.With(companyauth.RequirePermission("expenses.view")).Post(base+"/expenses/{expenseId}/manager-reject",m.handler.ManagerReject)
		sub.With(companyauth.RequirePermission("expenses.finalize")).Post(base+"/expenses/{expenseId}/accounting-approve",m.handler.AccountingApprove)
		sub.With(companyauth.RequirePermission("expenses.finalize")).Post(base+"/expenses/{expenseId}/accounting-reject",m.handler.AccountingReject)
		sub.Post(base+"/employees/{employeeId}/accountant",m.handler.GrantAccountant)
		sub.Delete(base+"/employees/{employeeId}/accountant",m.handler.RevokeAccountant)
	})
}
func (m *Module) Start(context.Context) error{return nil}
func (m *Module) Stop(context.Context) error{return nil}
var _ module.IModule=(*Module)(nil)
