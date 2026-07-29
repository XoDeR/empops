package place

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	placehttp "github.com/XoDeR/empops/api-go/internal/modules/place/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/geocoder"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for places.
type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *placehttp.Handler
}

// New creates an uninitialized place Module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "place" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee"}
}

func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.handler = placehttp.NewHandler(core.DB, geocoder.Noop{})
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Get("/countries", m.handler.ListCountries)

		sub.Route("/companies/{companyId}", func(co chi.Router) {
			co.Use(requireMember)
			co.Get("/employees/{employeeId}/places", m.handler.ListPlaces)
			co.Post("/employees/{employeeId}/places", m.handler.CreatePlace)
			co.Patch("/places/{placeId}", m.handler.UpdatePlace)
			co.Put("/places/{placeId}/activate", m.handler.ActivatePlace)
			co.Delete("/places/{placeId}", m.handler.DeletePlace)
		})
	})
}

func (m *Module) Start(ctx context.Context) error  { return nil }
func (m *Module) Stop(ctx context.Context) error   { return nil }

var _ module.IModule = (*Module)(nil)
