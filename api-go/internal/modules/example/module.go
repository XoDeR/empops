// Package example is a minimal vertical module demonstrating the modular
// DDD lifecycle (Initialize -> Start -> RegisterRoutes -> Stop). It only
// depends on pkg/*, never on Core's internal packages or other modules.
package example

import (
	"context"

	"github.com/go-chi/chi/v5"

	exhttp "github.com/XoDeR/empops/api-go/internal/modules/example/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

// Module implements module.IModule for the example vertical slice.
type Module struct {
	handler *exhttp.Handler
}

// New creates an uninitialized example Module.
func New() *Module {
	return &Module{}
}

// Name returns the module identifier used in config/modules.yaml.
func (m *Module) Name() string {
	return "example"
}

// Dependencies returns the other modules this module requires to be
// initialized first. The example module has none.
func (m *Module) Dependencies() []string {
	return nil
}

// Initialize wires the module's handlers using shared Core services.
func (m *Module) Initialize(ctx context.Context, core *module.Core) error {
	m.handler = exhttp.NewHandler()
	return nil
}

// RegisterRoutes mounts GET /example/ping.
func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/example", func(sub chi.Router) {
		sub.Get("/ping", m.handler.Ping)
	})
}

// Start begins background work. The example module has none.
func (m *Module) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down background work. The example module has none.
func (m *Module) Stop(ctx context.Context) error {
	return nil
}

var _ module.IModule = (*Module)(nil)
