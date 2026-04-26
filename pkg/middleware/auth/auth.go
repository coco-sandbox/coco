// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrInvalidKey   = errors.New("invalid API key")
	ErrKeyExpired   = errors.New("API key expired")
	ErrKeyDisabled  = errors.New("API key disabled")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type Role int32

const (
	RoleUnspecified Role = 0
	RoleAdmin       Role = 1
	RoleOperator    Role = 2
	RoleDeveloper   Role = 3
	RoleReadonly    Role = 4
)

type Scope int32

const (
	ScopeUnspecified   Scope = 0
	ScopeSandboxCreate Scope = 1
	ScopeSandboxRead   Scope = 2
	ScopeSandboxWrite  Scope = 3
	ScopeSandboxDelete Scope = 4
	ScopeTemplateRead  Scope = 5
	ScopeTemplateWrite Scope = 6
	ScopeClusterAdmin  Scope = 7
)

type APIKey struct {
	ID        string
	KeyHash   string
	Name      string
	Role      Role
	Scopes    []Scope
	CreatedBy string
	CreatedAt *timestamppb.Timestamp
	ExpiresAt *timestamppb.Timestamp
	Enabled   bool
}

type KeyStore interface {
	CreateKey(ctx context.Context, key *APIKey) (string, error)
	GetKey(ctx context.Context, id string) (*APIKey, error)
	GetKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	ListKeys(ctx context.Context) ([]*APIKey, error)
	DeleteKey(ctx context.Context, id string) error
	ValidateKey(ctx context.Context, rawKey string) (*APIKey, error)
}

type Authenticator struct {
	store KeyStore
}

func NewAuthenticator(store KeyStore) *Authenticator {
	return &Authenticator{store: store}
}

func (a *Authenticator) ValidateKey(ctx context.Context, rawKey string) (*APIKey, error) {
	hash := HashKey(rawKey)
	key, err := a.store.GetKeyByHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidKey
	}

	if !key.Enabled {
		return nil, ErrKeyDisabled
	}

	if key.ExpiresAt != nil {
		if time.Now().After(key.ExpiresAt.AsTime()) {
			return nil, ErrKeyExpired
		}
	}

	return key, nil
}

func (a *Authenticator) HasScope(key *APIKey, required Scope) bool {
	if key.Role == RoleAdmin {
		return true
	}

	for _, s := range key.Scopes {
		if s == required {
			return true
		}
		if s == ScopeUnspecified {
			return false
		}
	}

	return false
}

func (a *Authenticator) RequireScope(scopes ...Scope) func(ctx context.Context, key *APIKey) error {
	return func(ctx context.Context, key *APIKey) error {
		for _, required := range scopes {
			if !a.HasScope(key, required) {
				return ErrForbidden
			}
		}
		return nil
	}
}

func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

func GenerateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type ContextKey string

const (
	ContextKeyUser   ContextKey = "auth_user"
	ContextKeyKeyID  ContextKey = "auth_key_id"
	ContextKeyRole   ContextKey = "auth_role"
	ContextKeyScopes ContextKey = "auth_scopes"
)

func WithAuth(ctx context.Context, key *APIKey) context.Context {
	ctx = context.WithValue(ctx, ContextKeyUser, key.Name)
	ctx = context.WithValue(ctx, ContextKeyKeyID, key.ID)
	ctx = context.WithValue(ctx, ContextKeyRole, key.Role)
	ctx = context.WithValue(ctx, ContextKeyScopes, key.Scopes)
	return ctx
}

func UserFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUser).(string); ok {
		return v
	}
	return ""
}

func KeyIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyKeyID).(string); ok {
		return v
	}
	return ""
}

func RoleFromContext(ctx context.Context) Role {
	if v, ok := ctx.Value(ContextKeyRole).(Role); ok {
		return v
	}
	return RoleUnspecified
}

func ScopesFromContext(ctx context.Context) []Scope {
	if v, ok := ctx.Value(ContextKeyScopes).([]Scope); ok {
		return v
	}
	return nil
}
