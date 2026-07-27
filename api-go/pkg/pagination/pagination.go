// Package pagination provides a shared page/per-page request and response
// shape for list endpoints across Core and modules.
package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 15
	MaxPerPage     = 100
)

// Params is the parsed pagination request.
type Params struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// Offset returns the SQL OFFSET for these params.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the SQL LIMIT for these params.
func (p Params) Limit() int {
	return p.PerPage
}

// FromRequest reads "page" and "per_page" query parameters, applying
// sane defaults and bounds.
func FromRequest(r *http.Request) Params {
	page := parseIntDefault(r.URL.Query().Get("page"), DefaultPage)
	perPage := parseIntDefault(r.URL.Query().Get("per_page"), DefaultPerPage)

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{Page: page, PerPage: perPage}
}

func parseIntDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

// Meta describes the pagination metadata returned alongside a page of data.
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewMeta builds Meta from the request params and a total row count.
func NewMeta(p Params, total int64) Meta {
	totalPages := 0
	if p.PerPage > 0 {
		totalPages = int((total + int64(p.PerPage) - 1) / int64(p.PerPage))
	}
	return Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// Result wraps a page of items with its metadata.
type Result[T any] struct {
	Items []T  `json:"items"`
	Meta  Meta `json:"meta"`
}
