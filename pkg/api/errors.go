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

func WriteResourceExhausted(w http.ResponseWriter, message string) {
	WriteError(w, ErrResourceExhausted, message, "", http.StatusTooManyRequests)
}

func WriteAlreadyExists(w http.ResponseWriter, message string) {
	WriteError(w, ErrAlreadyExists, message, "", http.StatusConflict)
}

func WriteTimeout(w http.ResponseWriter, message string) {
	WriteError(w, ErrTimeout, message, "", http.StatusGatewayTimeout)
}

// WriteMethodNotAllowed responds with HTTP 405 wrapped in the spec error
// envelope. Spec/02 §5.1 has no dedicated logical code for method/path
// mismatches; we use INVALID_ARGUMENT, which fits the "wrong way to call
// this resource" semantic.
func WriteMethodNotAllowed(w http.ResponseWriter) {
	WriteError(w, ErrInvalidArgument, "method not allowed", "", http.StatusMethodNotAllowed)
}
