// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"coco/pkg/api"
	"coco/pkg/middleware/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

func timestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func timestampFromUnix(unix int64) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(unix, 0))
}

func extractID(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

type AuthHandler struct {
	store auth.KeyStore
}

func NewAuthHandler(store auth.KeyStore) *AuthHandler {
	return &AuthHandler{store: store}
}

type CreateAPIKeyRequest struct {
	Name      string       `json:"name"`
	Role      auth.Role    `json:"role"`
	Scopes    []auth.Scope `json:"scopes"`
	ExpiresAt int64        `json:"expires_at,omitempty"`
}

type APIKeyResponse struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Role      auth.Role    `json:"role"`
	Scopes    []auth.Scope `json:"scopes"`
	CreatedAt int64        `json:"created_at"`
	ExpiresAt int64        `json:"expires_at,omitempty"`
	Enabled   bool         `json:"enabled"`
}

func (h *AuthHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "invalid request body")
		return
	}

	rawKey, err := auth.GenerateKey()
	if err != nil {
		api.WriteInternalError(w, "failed to generate key")
		return
	}

	key := &auth.APIKey{
		ID:        generateID(),
		Name:      req.Name,
		Role:      req.Role,
		Scopes:    req.Scopes,
		KeyHash:   auth.HashKey(rawKey),
		CreatedAt: timestampNow(),
		Enabled:   true,
	}

	if req.ExpiresAt > 0 {
		key.ExpiresAt = timestampFromUnix(req.ExpiresAt)
	}

	if _, err := h.store.CreateKey(r.Context(), key); err != nil {
		api.WriteInternalError(w, "failed to create key")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Key    APIKeyResponse `json:"key"`
		RawKey string         `json:"raw_key"`
	}{
		Key: APIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Role:      key.Role,
			Scopes:    key.Scopes,
			CreatedAt: key.CreatedAt.AsTime().Unix(),
			ExpiresAt: key.ExpiresAt.AsTime().Unix(),
			Enabled:   key.Enabled,
		},
		RawKey: rawKey,
	})
}

func (h *AuthHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w)
		return
	}

	keys, err := h.store.ListKeys(r.Context())
	if err != nil {
		api.WriteInternalError(w, "failed to list keys")
		return
	}

	resp := make([]APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, APIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			Role:      key.Role,
			Scopes:    key.Scopes,
			CreatedAt: key.CreatedAt.AsTime().Unix(),
			ExpiresAt: key.ExpiresAt.AsTime().Unix(),
			Enabled:   key.Enabled,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Keys []APIKeyResponse `json:"keys"`
	}{
		Keys: resp,
	})
}

func (h *AuthHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.WriteMethodNotAllowed(w)
		return
	}

	path := r.URL.Path
	id := extractID(path, "/v1/api-keys/")

	if id == "" {
		api.WriteBadRequest(w, "missing key id")
		return
	}

	if err := h.store.DeleteKey(r.Context(), id); err != nil {
		api.WriteInternalError(w, "failed to delete key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "invalid request body")
		return
	}

	key, err := h.store.ValidateKey(r.Context(), req.Key)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Valid bool `json:"valid"`
		}{
			Valid: false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Valid bool           `json:"valid"`
		Key   APIKeyResponse `json:"key"`
	}{
		Valid: true,
		Key: APIKeyResponse{
			ID:     key.ID,
			Name:   key.Name,
			Role:   key.Role,
			Scopes: key.Scopes,
		},
	})
}
