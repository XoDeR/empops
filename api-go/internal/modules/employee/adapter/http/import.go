package http

import (
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type importRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportEmployees handles POST /companies/{companyId}/employees/import.
func (h *Handler) ImportEmployees(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid multipart form", nil)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		response.Fail(w, http.StatusBadRequest, "file is required", nil)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid CSV file", nil)
		return
	}

	colIndex := map[string]int{}
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	required := []string{"email", "first_name", "last_name"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			response.Fail(w, http.StatusBadRequest, "CSV must include headers: email, first_name, last_name", nil)
			return
		}
	}

	created := 0
	var importErrors []importRowError

	rowNum := 1
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		rowNum++
		if err != nil {
			importErrors = append(importErrors, importRowError{Row: rowNum, Message: "invalid CSV row"})
			continue
		}

		email := strings.TrimSpace(record[colIndex["email"]])
		firstName := strings.TrimSpace(record[colIndex["first_name"]])
		lastName := strings.TrimSpace(record[colIndex["last_name"]])
		if email == "" || firstName == "" || lastName == "" {
			importErrors = append(importErrors, importRowError{Row: rowNum, Message: "email, first_name and last_name are required"})
			continue
		}

		var hiredAt *time.Time
		if idx, ok := colIndex["hired_at"]; ok && idx < len(record) {
			v := strings.TrimSpace(record[idx])
			if v != "" {
				t, err := time.Parse("2006-01-02", v)
				if err != nil {
					importErrors = append(importErrors, importRowError{Row: rowNum, Message: "invalid hired_at"})
					continue
				}
				hiredAt = &t
			}
		}

		var positionID *string
		if idx, ok := colIndex["position_id"]; ok && idx < len(record) {
			v := strings.TrimSpace(record[idx])
			if v != "" {
				positionID = &v
			}
		}

		employeeID := uuidv7.New()
		var id string
		err = h.pool.QueryRow(r.Context(), `
			INSERT INTO employees (
				id, company_id, user_id, email, first_name, last_name, hired_at,
				position_id, employee_status_id, locked, created_at, updated_at
			) VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, NULL, false, now(), now())
			RETURNING id`,
			employeeID, companyID, email, firstName, lastName, hiredAt, positionID,
		).Scan(&id)
		if err != nil {
			if isUniqueViolation(err) {
				importErrors = append(importErrors, importRowError{Row: rowNum, Message: "email already exists"})
			} else {
				importErrors = append(importErrors, importRowError{Row: rowNum, Message: "create failed"})
			}
			continue
		}

		if err := assignRole(r.Context(), h.pool, id, "employee"); err != nil {
			importErrors = append(importErrors, importRowError{Row: rowNum, Message: "role assignment failed"})
			continue
		}

		created++
	}

	if importErrors == nil {
		importErrors = []importRowError{}
	}

	response.OK(w, "", map[string]interface{}{
		"created": created,
		"errors":  importErrors,
	})
}
