// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package types

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleAgent    Role = "agent"
	RoleReadonly Role = "readonly"
)

type APIKey struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	Roles     []Role    `json:"roles"`
	Expires   int64     `json:"expires,omitempty"`
}
