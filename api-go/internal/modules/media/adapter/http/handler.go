package http

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/file-uploads-go/backend/pkg/upload/chunked"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

const (
	modelTemporaryUpload = "temporary_upload"
	modelEmployee        = "employee"
	modelCompany         = "company"
)

// Handler handles media attach and file serving.
type Handler struct {
	pool      *pgxpool.Pool
	uploadDir string
}

// NewHandler constructs a media Handler.
func NewHandler(pool *pgxpool.Pool, uploadDir string) *Handler {
	return &Handler{pool: pool, uploadDir: uploadDir}
}

// WrapComplete registers uploaded files in the media table (upload-lib compatible JSON).
func (h *Handler) WrapComplete(cm *chunked.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		cm.CompleteUpload(rec, r)
		if rec.Code != http.StatusOK {
			w.WriteHeader(rec.Code)
			_, _ = io.Copy(w, rec.Body)
			return
		}

		var result map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			http.Error(w, "Invalid response", http.StatusInternalServerError)
			return
		}

		filename, _ := result["filename"].(string)
		size, _ := result["size"].(float64)
		tempID, mediaID, err := h.registerTemporaryMedia(r, filename, int64(size))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result["temporary_upload_id"] = tempID
		result["media_id"] = mediaID

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func (h *Handler) registerTemporaryMedia(r *http.Request, filename string, size int64) (int64, int64, error) {
	var tempID, mediaID int64
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO temporary_uploads DEFAULT VALUES RETURNING id`,
	).Scan(&tempID)
	if err != nil {
		return 0, 0, err
	}

	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO media (model_type, model_id, collection_name, name, file_name, mime_type, size)
		VALUES ($1, $2, 'uploads', $3, $3, $4, $5)
		RETURNING id`,
		modelTemporaryUpload, strconv.FormatInt(tempID, 10), filename, mimeType, size,
	).Scan(&mediaID)
	if err != nil {
		return 0, 0, err
	}

	return tempID, mediaID, nil
}

type attachMediaRequest struct {
	TemporaryUploadID int64 `json:"temporary_upload_id"`
	MediaID           int64 `json:"media_id"`
}

// AttachEmployeeAvatar handles PUT /companies/{companyId}/employees/{employeeId}/avatar.
func (h *Handler) AttachEmployeeAvatar(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
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
	if member.EmployeeID != employeeID && !member.HasPermission("employees.update") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req attachMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	if err := h.attachMedia(r, modelEmployee, employeeID, "avatar", req); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Media not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "attach avatar failed", err.Error())
		return
	}

	avatarURL := fmt.Sprintf("/api/v1/media/%d/file", req.MediaID)
	response.OK(w, "Avatar updated", map[string]interface{}{
		"avatar_url": avatarURL,
	})
}

// AttachCompanyLogo handles PUT /companies/{companyId}/logo.
func (h *Handler) AttachCompanyLogo(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	var req attachMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	if err := h.attachMedia(r, modelCompany, companyID, "logo", req); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Media not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "attach logo failed", err.Error())
		return
	}

	logoURL := fmt.Sprintf("/api/v1/media/%d/file", req.MediaID)
	response.OK(w, "Logo updated", map[string]interface{}{
		"logo_url": logoURL,
	})
}

func (h *Handler) attachMedia(r *http.Request, modelType, modelID, collection string, req attachMediaRequest) error {
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())

	var fileName string
	err = tx.QueryRow(r.Context(), `
		SELECT file_name FROM media
		WHERE id = $1 AND model_type = $2 AND model_id = $3`,
		req.MediaID, modelTemporaryUpload, strconv.FormatInt(req.TemporaryUploadID, 10),
	).Scan(&fileName)
	if err != nil {
		return err
	}

	_, err = tx.Exec(r.Context(), `
		DELETE FROM media WHERE model_type = $1 AND model_id = $2 AND collection_name = $3`,
		modelType, modelID, collection,
	)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(r.Context(), `
		UPDATE media SET model_type = $2, model_id = $3, collection_name = $4, updated_at = now()
		WHERE id = $1 AND model_type = $5 AND model_id = $6`,
		req.MediaID, modelType, modelID, collection,
		modelTemporaryUpload, strconv.FormatInt(req.TemporaryUploadID, 10),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(r.Context())
}

// ServeFile handles GET /media/{mediaId}/file.
func (h *Handler) ServeFile(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaId")
	var fileName, mimeType string
	err := h.pool.QueryRow(r.Context(), `
		SELECT file_name, COALESCE(mime_type, 'application/octet-stream')
		FROM media WHERE id = $1`, mediaID,
	).Scan(&fileName, &mimeType)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "File not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}

	path := filepath.Join(h.uploadDir, fileName)
	f, err := os.Open(path)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "File not found on disk", nil)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", mimeType)
	modTime := time.Time{}
	if info, err := os.Stat(path); err == nil {
		modTime = info.ModTime()
	}
	http.ServeContent(w, r, fileName, modTime, f)
}

func (h *Handler) employeeInCompany(r *http.Request, companyID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND company_id = $2)`,
		employeeID, companyID,
	).Scan(&exists)
	return exists
}
