// Package http contains the example module's Chi handlers.
package http

import (
	"net/http"

	"github.com/XoDeR/empops/api-go/pkg/response"
)

// Handler serves the example module's HTTP routes.
type Handler struct{}

// NewHandler builds an example Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Ping handles GET /example/ping.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "pong", map[string]bool{"pong": true})
}
