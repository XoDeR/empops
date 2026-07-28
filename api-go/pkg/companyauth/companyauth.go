// Package companyauth provides company-scoped membership and permission
// middleware shared by the company and employee modules.
package companyauth

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

type contextKey string

const memberContextKey contextKey = "companyMember"

// Member holds the authenticated user's membership in a company route.
type Member struct {
	CompanyID  string
	EmployeeID string
	UserID     string
	Roles      []string
	Perms      map[string]bool
}

// MemberFromContext returns membership loaded by RequireMember middleware.
func MemberFromContext(ctx context.Context) (Member, bool) {
	m, ok := ctx.Value(memberContextKey).(Member)
	return m, ok
}

// HasPermission checks whether the member has a named permission.
func (m Member) HasPermission(name string) bool {
	return m.Perms[name]
}

// RequireMember loads company membership for routes with {companyId}.
func RequireMember(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := httpauth.UserIDFromContext(r.Context())
			if !ok {
				response.Fail(w, http.StatusUnauthorized, "Unauthenticated", nil)
				return
			}

			companyID := chi.URLParam(r, "companyId")
			if companyID == "" {
				response.Fail(w, http.StatusBadRequest, "companyId required", nil)
				return
			}

			var member Member
			var locked bool
			err := pool.QueryRow(r.Context(), `
				SELECT e.id, e.company_id, e.user_id, e.locked
				FROM employees e
				WHERE e.company_id = $1 AND e.user_id = $2`, companyID, userID,
			).Scan(&member.EmployeeID, &member.CompanyID, &member.UserID, &locked)
			if err == pgx.ErrNoRows {
				response.Fail(w, http.StatusForbidden, "Not a member of this company", nil)
				return
			}
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "membership lookup failed", err.Error())
				return
			}
			if locked {
				response.Fail(w, http.StatusForbidden, "Not a member of this company", nil)
				return
			}

			rows, err := pool.Query(r.Context(), `
				SELECT r.name FROM employee_roles er
				JOIN roles r ON r.id = er.role_id
				WHERE er.employee_id = $1 ORDER BY r.name`, member.EmployeeID)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "roles lookup failed", err.Error())
				return
			}
			defer rows.Close()
			for rows.Next() {
				var role string
				if err := rows.Scan(&role); err != nil {
					response.Fail(w, http.StatusInternalServerError, "roles scan failed", err.Error())
					return
				}
				member.Roles = append(member.Roles, role)
			}

			permRows, err := pool.Query(r.Context(), `
				SELECT DISTINCT p.name FROM employee_roles er
				JOIN role_permissions rp ON rp.role_id = er.role_id
				JOIN permissions p ON p.id = rp.permission_id
				WHERE er.employee_id = $1`, member.EmployeeID)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "permissions lookup failed", err.Error())
				return
			}
			defer permRows.Close()
			member.Perms = make(map[string]bool)
			for permRows.Next() {
				var perm string
				if err := permRows.Scan(&perm); err != nil {
					response.Fail(w, http.StatusInternalServerError, "permissions scan failed", err.Error())
					return
				}
				member.Perms[perm] = true
			}

			ctx := context.WithValue(r.Context(), memberContextKey, member)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission rejects requests when the member lacks any listed permission.
func RequirePermission(perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			member, ok := MemberFromContext(r.Context())
			if !ok {
				response.Fail(w, http.StatusForbidden, "Company membership required", nil)
				return
			}
			for _, p := range perms {
				if member.HasPermission(p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		})
	}
}
