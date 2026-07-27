// Package response defines the shared JSON envelope used by every HTTP
// handler in Core and in the vertical modules, mirroring the Laravel API
// contract so both backends stay wire-compatible.
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// Envelope is the standard API response shape:
//
//	{ "success": true, "message": "...", "data": {}, "error": null, "timestamp": "ISO8601" }
type Envelope struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Error     interface{} `json:"error"`
	Timestamp string      `json:"timestamp"`
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// OK writes a successful envelope with HTTP 200.
func OK(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, Envelope{
		Success:   true,
		Message:   message,
		Data:      data,
		Error:     nil,
		Timestamp: now(),
	})
}

// Created writes a successful envelope with HTTP 201.
func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, Envelope{
		Success:   true,
		Message:   message,
		Data:      data,
		Error:     nil,
		Timestamp: now(),
	})
}

// Fail writes a failed envelope with the given HTTP status code.
func Fail(w http.ResponseWriter, status int, message string, err interface{}) {
	JSON(w, status, Envelope{
		Success:   false,
		Message:   message,
		Data:      nil,
		Error:     err,
		Timestamp: now(),
	})
}

// JSON writes any envelope value with the given status code.
func JSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
