package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/geocoder"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

const employeePlacableType = "employee"

// Handler serves place and country HTTP endpoints.
type Handler struct {
	pool     *pgxpool.Pool
	geocoder geocoder.Geocoder
}

// NewHandler constructs a place Handler.
func NewHandler(pool *pgxpool.Pool, geo geocoder.Geocoder) *Handler {
	return &Handler{pool: pool, geocoder: geo}
}

// ListCountries handles GET /countries.
func (h *Handler) ListCountries(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name, code FROM countries ORDER BY name`)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list countries failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		var code *string
		if err := rows.Scan(&id, &name, &code); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id": id, "name": name, "code": code,
		})
	}
	response.OK(w, "", list)
}

// ListPlaces handles GET /companies/{companyId}/employees/{employeeId}/places.
func (h *Handler) ListPlaces(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.employeeInCompany(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if !canViewPlaces(actor, employeeID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT p.id, p.street, p.city, p.province, p.postal_code,
			p.country_id, c.name, c.code, p.latitude, p.longitude, p.is_active
		FROM places p
		LEFT JOIN countries c ON c.id = p.country_id
		WHERE p.placable_type = $1 AND p.placable_id = $2
		ORDER BY p.is_active DESC, p.created_at DESC`,
		employeePlacableType, employeeID,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list places failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		payload, err := scanPlaceRow(rows)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

type createPlaceRequest struct {
	Street     *string  `json:"street"`
	City       *string  `json:"city"`
	Province   *string  `json:"province"`
	PostalCode *string  `json:"postal_code"`
	CountryID  *string  `json:"country_id"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	IsActive   *bool    `json:"is_active"`
}

// CreatePlace handles POST /companies/{companyId}/employees/{employeeId}/places.
func (h *Handler) CreatePlace(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.employeeInCompany(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if !canCreatePlace(actor, employeeID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req createPlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	isActive := req.IsActive != nil && *req.IsActive
	coords, err := h.resolveCoords(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadGateway, "Geocoding failed", err.Error())
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	if isActive {
		_, _ = tx.Exec(r.Context(), `
			UPDATE places SET is_active = false, updated_at = now()
			WHERE placable_type = $1 AND placable_id = $2`,
			employeePlacableType, employeeID,
		)
	}

	placeID := uuidv7.New()
	_, err = tx.Exec(r.Context(), `
		INSERT INTO places (
			id, placable_id, placable_type, street, city, province, postal_code,
			country_id, latitude, longitude, is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now(), now())`,
		placeID, employeeID, employeePlacableType,
		req.Street, req.City, req.Province, req.PostalCode, req.CountryID,
		coords.Latitude, coords.Longitude, isActive,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create place failed", err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	payload, err := h.placePayload(r.Context(), placeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place payload failed", err.Error())
		return
	}
	response.Created(w, "Place created", payload)
}

type updatePlaceRequest struct {
	Street     *string  `json:"street"`
	City       *string  `json:"city"`
	Province   *string  `json:"province"`
	PostalCode *string  `json:"postal_code"`
	CountryID  *string  `json:"country_id"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	IsActive   *bool    `json:"is_active"`
}

// UpdatePlace handles PATCH /companies/{companyId}/places/{placeId}.
func (h *Handler) UpdatePlace(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	placeID := chi.URLParam(r, "placeId")

	employeeID, err := h.placeEmployeeID(r.Context(), companyID, placeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Place not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place lookup failed", err.Error())
		return
	}
	if !canManagePlaces(actor, employeeID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req updatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var street, city, province, postalCode *string
	var countryID *string
	var lat, lng *float64
	var isActive bool
	err = h.pool.QueryRow(r.Context(), `
		SELECT street, city, province, postal_code, country_id, latitude, longitude, is_active
		FROM places WHERE id = $1`, placeID,
	).Scan(&street, &city, &province, &postalCode, &countryID, &lat, &lng, &isActive)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place lookup failed", err.Error())
		return
	}

	if req.Street != nil {
		street = req.Street
	}
	if req.City != nil {
		city = req.City
	}
	if req.Province != nil {
		province = req.Province
	}
	if req.PostalCode != nil {
		postalCode = req.PostalCode
	}
	if req.CountryID != nil {
		countryID = req.CountryID
	}
	if req.Latitude != nil || req.Longitude != nil {
		lat, lng = req.Latitude, req.Longitude
	} else if req.Street != nil || req.City != nil || req.Province != nil || req.PostalCode != nil || req.CountryID != nil {
		coords, geoErr := h.resolveCoords(r.Context(), createPlaceRequest{
			Street: street, City: city, Province: province, PostalCode: postalCode, CountryID: countryID,
		})
		if geoErr != nil {
			response.Fail(w, http.StatusBadGateway, "Geocoding failed", geoErr.Error())
			return
		}
		lat, lng = coords.Latitude, coords.Longitude
	}
	if req.IsActive != nil && *req.IsActive {
		_, _ = h.pool.Exec(r.Context(), `
			UPDATE places SET is_active = false, updated_at = now()
			WHERE placable_type = $1 AND placable_id = $2 AND id != $3`,
			employeePlacableType, employeeID, placeID,
		)
		isActive = true
	} else if req.IsActive != nil {
		isActive = *req.IsActive
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE places SET street=$2, city=$3, province=$4, postal_code=$5, country_id=$6,
			latitude=$7, longitude=$8, is_active=$9, updated_at=now()
		WHERE id=$1`,
		placeID, street, city, province, postalCode, countryID, lat, lng, isActive,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update place failed", err.Error())
		return
	}

	payload, err := h.placePayload(r.Context(), placeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place payload failed", err.Error())
		return
	}
	response.OK(w, "Place updated", payload)
}

// ActivatePlace handles PUT /companies/{companyId}/places/{placeId}/activate.
func (h *Handler) ActivatePlace(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	placeID := chi.URLParam(r, "placeId")

	employeeID, err := h.placeEmployeeID(r.Context(), companyID, placeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Place not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place lookup failed", err.Error())
		return
	}
	if !canManagePlaces(actor, employeeID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	_, _ = h.pool.Exec(r.Context(), `
		UPDATE places SET is_active = false, updated_at = now()
		WHERE placable_type = $1 AND placable_id = $2`,
		employeePlacableType, employeeID,
	)
	_, err = h.pool.Exec(r.Context(), `
		UPDATE places SET is_active = true, updated_at = now() WHERE id = $1`, placeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "activate place failed", err.Error())
		return
	}

	payload, err := h.placePayload(r.Context(), placeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place payload failed", err.Error())
		return
	}
	response.OK(w, "Place activated", payload)
}

// DeletePlace handles DELETE /companies/{companyId}/places/{placeId}.
func (h *Handler) DeletePlace(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	placeID := chi.URLParam(r, "placeId")

	employeeID, err := h.placeEmployeeID(r.Context(), companyID, placeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Place not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "place lookup failed", err.Error())
		return
	}
	if !canManagePlaces(actor, employeeID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `DELETE FROM places WHERE id = $1`, placeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete place failed", err.Error())
		return
	}
	response.OK(w, "Place deleted", nil)
}

func (h *Handler) employeeInCompany(r *http.Request, companyID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND company_id = $2)`,
		employeeID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) placeEmployeeID(ctx context.Context, companyID, placeID string) (string, error) {
	var employeeID string
	err := h.pool.QueryRow(ctx, `
		SELECT p.placable_id FROM places p
		JOIN employees e ON e.id = p.placable_id
		WHERE p.id = $1 AND p.placable_type = $2 AND e.company_id = $3`,
		placeID, employeePlacableType, companyID,
	).Scan(&employeeID)
	return employeeID, err
}

func (h *Handler) placePayload(ctx context.Context, placeID string) (map[string]interface{}, error) {
	row := h.pool.QueryRow(ctx, `
		SELECT p.id, p.street, p.city, p.province, p.postal_code,
			p.country_id, c.name, c.code, p.latitude, p.longitude, p.is_active
		FROM places p
		LEFT JOIN countries c ON c.id = p.country_id
		WHERE p.id = $1`, placeID)
	return scanPlaceRow(row)
}

func scanPlaceRow(row pgx.Row) (map[string]interface{}, error) {
	var id string
	var street, city, province, postalCode *string
	var countryID, countryName, countryCode *string
	var lat, lng *float64
	var isActive bool
	if err := row.Scan(&id, &street, &city, &province, &postalCode,
		&countryID, &countryName, &countryCode, &lat, &lng, &isActive); err != nil {
		return nil, err
	}
	var country interface{}
	if countryID != nil && countryName != nil {
		country = map[string]interface{}{
			"id":   *countryID,
			"name": *countryName,
			"code": countryCode,
		}
	}
	return map[string]interface{}{
		"id":          id,
		"street":      street,
		"city":        city,
		"province":    province,
		"postal_code": postalCode,
		"country":     country,
		"latitude":    lat,
		"longitude":   lng,
		"is_active":   isActive,
	}, nil
}

func (h *Handler) resolveCoords(ctx context.Context, req createPlaceRequest) (geocoder.Coordinates, error) {
	var countryName string
	if req.CountryID != nil && *req.CountryID != "" {
		_ = h.pool.QueryRow(ctx, `SELECT name FROM countries WHERE id = $1`, *req.CountryID).Scan(&countryName)
	}
	addr := geocoder.Address{
		Street:     strVal(req.Street),
		City:       strVal(req.City),
		Province:   strVal(req.Province),
		PostalCode: strVal(req.PostalCode),
		Country:    countryName,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
	}
	return h.geocoder.Geocode(ctx, addr)
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func canViewPlaces(actor companyauth.Member, employeeID string) bool {
	return actor.EmployeeID == employeeID || actor.HasPermission("places.view")
}

func canCreatePlace(actor companyauth.Member, employeeID string) bool {
	return actor.EmployeeID == employeeID || actor.HasPermission("places.create")
}

func canManagePlaces(actor companyauth.Member, employeeID string) bool {
	return actor.EmployeeID == employeeID || actor.HasPermission("places.update")
}
