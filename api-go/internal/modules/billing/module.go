// Package billing exposes paid-plan invoice APIs when explicitly enabled.
package billing

import (
	"context"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	billinghttp "github.com/XoDeR/empops/api-go/internal/modules/billing/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct{pool *pgxpool.Pool;jwt *jwt.Manager;handler *billinghttp.Handler;enabled bool}
func New()*Module{return &Module{}}
func(m *Module)Name()string{return "billing"}
func(m *Module)Dependencies()[]string{return []string{"company","employee"}}
func(m *Module)Initialize(_ context.Context,core *module.Core)error{m.pool,m.jwt=core.DB,core.JWT;m.handler=billinghttp.NewHandler(core.DB);m.enabled,_=strconv.ParseBool(os.Getenv("ENABLE_PAID_PLAN"));return nil}
func(m *Module)RegisterRoutes(r chi.Router){if !m.enabled{return};r.Group(func(sub chi.Router){sub.Use(httpauth.RequireAuth(m.jwt));sub.Use(companyauth.RequireMember(m.pool));sub.With(companyauth.RequirePermission("billing.view")).Get("/companies/{companyId}/invoices",m.handler.ListInvoices)})}
func(m *Module)Start(context.Context)error{return nil};func(m *Module)Stop(context.Context)error{return nil}
var _ module.IModule=(*Module)(nil)
