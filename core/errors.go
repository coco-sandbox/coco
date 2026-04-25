// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"net/http"
)

// =============================================================================
// Error Codes
// =============================================================================

type ErrorCode string

const (
	ErrCodeSandboxNotFound   ErrorCode = "SANDBOX_NOT_FOUND"
	ErrCodeCheckpointNotFound ErrorCode = "CHECKPOINT_NOT_FOUND"
	ErrCodeReplayNotFound     ErrorCode = "REPLAY_NOT_FOUND"
	ErrCodeInvalidState       ErrorCode = "INVALID_STATE"
	ErrCodeInvalidRequest     ErrorCode = "INVALID_REQUEST"
	ErrCodeExecTimeout        ErrorCode = "EXEC_TIMEOUT"
	ErrCodeForkDepthExceeded  ErrorCode = "FORK_DEPTH_EXCEEDED"
	ErrCodeHibernateFailed    ErrorCode = "HIBERNATE_FAILED"
	ErrCodeResumeFailed       ErrorCode = "RESUME_FAILED"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeRateLimited        ErrorCode = "RATE_LIMITED"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
)

// =============================================================================
// Error Response
// =============================================================================

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// =============================================================================
// Error Helper Functions
// =============================================================================

func newErrorResponse(code ErrorCode, message string, details string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// =============================================================================
// Validation Errors (400)
// =============================================================================

func writeValidationError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, newErrorResponse(ErrCodeInvalidRequest, message, ""))
}

func writeInvalidStateError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, newErrorResponse(ErrCodeInvalidState, message, ""))
}

func writeForkDepthError(w http.ResponseWriter, maxDepth int) {
	writeJSON(w, http.StatusBadRequest, newErrorResponse(
		ErrCodeForkDepthExceeded,
		fmt.Sprintf("Fork depth exceeds maximum of %d", maxDepth),
		"",
	))
}

// =============================================================================
// Not Found Errors (404)
// =============================================================================

func writeSandboxNotFoundError(w http.ResponseWriter, id string) {
	writeJSON(w, http.StatusNotFound, newErrorResponse(
		ErrCodeSandboxNotFound,
		fmt.Sprintf("Sandbox %s not found", id),
		"",
	))
}

func writeCheckpointNotFoundError(w http.ResponseWriter, id string) {
	writeJSON(w, http.StatusNotFound, newErrorResponse(
		ErrCodeCheckpointNotFound,
		fmt.Sprintf("Checkpoint %s not found", id),
		"",
	))
}

func writeReplayNotFoundError(w http.ResponseWriter, id string) {
	writeJSON(w, http.StatusNotFound, newErrorResponse(
		ErrCodeReplayNotFound,
		fmt.Sprintf("Replay %s not found", id),
		"",
	))
}

// =============================================================================
// Conflict Errors (409)
// =============================================================================

func writeConflictError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, newErrorResponse(ErrCodeInvalidState, message, ""))
}

// =============================================================================
// Rate Limit Errors (429)
// =============================================================================

func writeRateLimitedError(w http.ResponseWriter, retryAfter int) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	writeJSON(w, http.StatusTooManyRequests, newErrorResponse(
		ErrCodeRateLimited,
		fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter),
		"",
	))
}

// =============================================================================
// Unauthorized Errors (401/403)
// =============================================================================

func writeUnauthorizedError(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, newErrorResponse(
		ErrCodeUnauthorized,
		"Authentication required",
		"",
	))
}

func writeForbiddenError(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, newErrorResponse(
		ErrCodeUnauthorized,
		"Insufficient permissions",
		"",
	))
}

// =============================================================================
// Internal Errors (500)
// =============================================================================

func writeInternalError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, newErrorResponse(
		ErrCodeInternalError,
		message,
		"",
	))
}

func writeExecTimeoutError(w http.ResponseWriter, timeoutSeconds int) {
	writeJSON(w, http.StatusGatewayTimeout, newErrorResponse(
		ErrCodeExecTimeout,
		fmt.Sprintf("Command execution timed out after %d seconds", timeoutSeconds),
		"",
	))
}

func writeHibernateError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, newErrorResponse(
		ErrCodeHibernateFailed,
		message,
		"",
	))
}

func writeResumeError(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, newErrorResponse(
		ErrCodeResumeFailed,
		message,
		"",
	))
}