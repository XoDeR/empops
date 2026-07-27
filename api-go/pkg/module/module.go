// Package module defines the modular-DDD registry: an IModule lifecycle
// (Initialize -> Start -> RegisterRoutes -> Stop) plus a Core facade that
// gives modules access to shared platform services (config, logger, JWT,
// bus, DB) without letting them import Core's internal packages.
package module

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/bus"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
)

// Core is the facade injected into every module so it can reach shared
// platform services without importing Core's internal packages. Config is
// intentionally omitted here (pkg must never import internal/*); modules
// that need config values receive them explicitly during wiring in cmd/api.
type Core struct {
	Logger *slog.Logger
	JWT    *jwt.Manager
	Bus    bus.Bus
	DB     *pgxpool.Pool // may be nil in Step 0 (stub, no database wired yet)
}

// IModule is the lifecycle every vertical module (and Core itself) implements.
type IModule interface {
	// Name returns the unique module identifier used in config/modules.yaml.
	Name() string
	// Dependencies lists module names that must be initialized before this one.
	Dependencies() []string
	// Initialize wires repositories, use cases and handlers using core.
	Initialize(ctx context.Context, core *Core) error
	// RegisterRoutes mounts the module's HTTP routes on the shared router.
	RegisterRoutes(r chi.Router)
	// Start begins any background work (schedulers, consumers). No-op for most modules.
	Start(ctx context.Context) error
	// Stop gracefully shuts down background work.
	Stop(ctx context.Context) error
}

// Registry holds every module registered via init() and drives its lifecycle.
type Registry struct {
	modules map[string]IModule
	order   []string
}

// DefaultRegistry is the process-wide registry modules register themselves into.
var DefaultRegistry = NewRegistry()

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]IModule)}
}

// Register adds a module definition. Call from an init() function in each
// module's register.go so blank-importing the module package is enough to
// make it available.
func (r *Registry) Register(m IModule) {
	if _, exists := r.modules[m.Name()]; exists {
		panic(fmt.Sprintf("module: %q already registered", m.Name()))
	}
	r.modules[m.Name()] = m
	r.order = append(r.order, m.Name())
}

// InitializeAll initializes every module named in enabled, respecting each
// module's Dependencies() ordering.
func (r *Registry) InitializeAll(ctx context.Context, core *Core, enabled []string) ([]IModule, error) {
	ordered, err := r.resolveOrder(enabled)
	if err != nil {
		return nil, err
	}

	initialized := make([]IModule, 0, len(ordered))
	for _, name := range ordered {
		m := r.modules[name]
		if err := m.Initialize(ctx, core); err != nil {
			return nil, fmt.Errorf("module %q: initialize: %w", name, err)
		}
		initialized = append(initialized, m)
	}
	return initialized, nil
}

// RegisterRoutes mounts every initialized module's routes on r.
func RegisterRoutes(router chi.Router, modules []IModule) {
	for _, m := range modules {
		m.RegisterRoutes(router)
	}
}

// StartAll starts every initialized module.
func StartAll(ctx context.Context, modules []IModule) error {
	for _, m := range modules {
		if err := m.Start(ctx); err != nil {
			return fmt.Errorf("module %q: start: %w", m.Name(), err)
		}
	}
	return nil
}

// StopAll stops every initialized module in reverse order.
func StopAll(ctx context.Context, modules []IModule) {
	for i := len(modules) - 1; i >= 0; i-- {
		_ = modules[i].Stop(ctx)
	}
}

func (r *Registry) resolveOrder(enabled []string) ([]string, error) {
	enabledSet := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		enabledSet[name] = true
	}

	var resolved []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("module: circular dependency detected at %q", name)
		}
		m, ok := r.modules[name]
		if !ok {
			return fmt.Errorf("module: %q enabled but not registered", name)
		}

		visiting[name] = true
		for _, dep := range m.Dependencies() {
			if !enabledSet[dep] {
				return fmt.Errorf("module: %q depends on %q which is not enabled", name, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		resolved = append(resolved, name)
		return nil
	}

	for _, name := range enabled {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}
