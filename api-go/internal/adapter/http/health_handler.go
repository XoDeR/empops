// Package httpadapter contains Core's Chi router, handlers and DTO wiring.
// Named "httpadapter" (not "http") so callers can import both this package
// and the standard library net/http without aliasing.
package httpadapter

import (
	"net/http"

	"github.com/XoDeR/empops/api-go/pkg/response"
)

// AppVersion and AppName are reported by GET /version.
const (
	AppVersion = "0.0.0"
	AppName    = "empops-go"
)

// HealthHandler serves GET /health.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "ok", map[string]string{"status": "ok"})
}

// VersionHandler serves GET /version.
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "ok", map[string]string{
		"version": AppVersion,
		"name":    AppName,
	})
}
