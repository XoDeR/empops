package media

import (
	"context"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	mediahttp "github.com/XoDeR/empops/api-go/internal/modules/media/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for media attach and file serving.
type Module struct {
	pool      *pgxpool.Pool
	jwt       *jwt.Manager
	handler   *mediahttp.Handler
	uploadDir string
}

// New creates an uninitialized media Module.
func New() *Module {
	return &Module{}
}

func (m *Module) Name() string { return "media" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee"}
}

func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.pool = core.DB
	m.jwt = core.JWT
	m.uploadDir = os.Getenv("EMPOPS_UPLOAD_DIR")
	if m.uploadDir == "" {
		m.uploadDir = "./uploads"
	}
	m.handler = mediahttp.NewHandler(core.DB, m.uploadDir)
	return nil
}

// Handler exposes the HTTP handler (e.g. for upload complete wrapping in main).
func (m *Module) Handler() *mediahttp.Handler {
	return m.handler
}

func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Get("/media/{mediaId}/file", m.handler.ServeFile)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)

		sub.Put("/companies/{companyId}/employees/{employeeId}/avatar", m.handler.AttachEmployeeAvatar)
		sub.With(companyauth.RequirePermission("company.update")).
			Put("/companies/{companyId}/logo", m.handler.AttachCompanyLogo)
	})
}

func (m *Module) Start(ctx context.Context) error  { return nil }
func (m *Module) Stop(ctx context.Context) error   { return nil }

var _ module.IModule = (*Module)(nil)
