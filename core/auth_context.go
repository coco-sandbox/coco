// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
)

// Context keys for auth and tenant info
type contextKey string

const (
	apiKeyContextKey  contextKey = "api_key"
	tenantIDContextKey contextKey = "tenant_id"
)

// WithAPIKey returns a new context with the API key set
func WithAPIKey(ctx context.Context, apiKey *APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, apiKey)
}

// APIKeyFromContext retrieves the API key from context
func APIKeyFromContext(ctx context.Context) *APIKey {
	if apiKey, ok := ctx.Value(apiKeyContextKey).(*APIKey); ok {
		return apiKey
	}
	return nil
}

// WithTenantID returns a new context with the tenant ID set
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDContextKey, tenantID)
}

// TenantIDFromContext retrieves the tenant ID from context
func TenantIDFromContext(ctx context.Context) string {
	if tenantID, ok := ctx.Value(tenantIDContextKey).(string); ok {
		return tenantID
	}
	return ""
}