package api

import (
	"encoding/json"
	"net/http"
)

type ErrorCode string

const (
	ErrNotFound          ErrorCode = "NOT_FOUND"
	ErrAlreadyExists     ErrorCode = "ALREADY_EXISTS"
	ErrInvalidArgument   ErrorCode = "INVALID_ARGUMENT"
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrInternal          ErrorCode = "INTERNAL"
	ErrUnavailable       ErrorCode = "UNAVAILABLE"
	ErrTimeout           ErrorCode = "TIMEOUT"
)

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func WriteError(w http.ResponseWriter, code ErrorCode, message string, details string, status int) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, ErrNotFound, message, "", http.StatusNotFound)
}

func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, ErrInvalidArgument, message, "", http.StatusBadRequest)
}

func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteError(w, ErrPermissionDenied, message, "", http.StatusUnauthorized)
}

func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, ErrInternal, message, "", http.StatusInternalServerError)
}

func WriteServiceUnavailable(w http.ResponseWriter, message string) {
	WriteError(w, ErrUnavailable, message, "", http.StatusServiceUnavailable)
}
