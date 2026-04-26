// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package rbac

import (
	"net/http"

	"coco/pkg/api"
	"coco/pkg/middleware/auth"
)

type Policy struct {
	allow map[auth.Scope][]string
	deny  map[auth.Scope][]string
}

type Enforcer struct {
	policies map[string]*Policy
}

func NewEnforcer() *Enforcer {
	return &Enforcer{
		policies: make(map[string]*Policy),
	}
}

func (e *Enforcer) AddPolicy(subject string, scope auth.Scope, resources ...string) {
	policy, ok := e.policies[subject]
	if !ok {
		policy = &Policy{
			allow: make(map[auth.Scope][]string),
			deny:  make(map[auth.Scope][]string),
		}
		e.policies[subject] = policy
	}

	policy.allow[scope] = append(policy.allow[scope], resources...)
}

func (e *Enforcer) Enforce(scope auth.Scope, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			role := auth.RoleFromContext(ctx)

			if role == auth.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			keyID := auth.KeyIDFromContext(ctx)
			policy, ok := e.policies[keyID]
			if !ok {
				api.WriteError(w, api.ErrPermissionDenied, "no policy defined", "", http.StatusForbidden)
				return
			}

			allowed, exists := policy.allow[scope]
			if !exists || len(allowed) == 0 {
				api.WriteError(w, api.ErrPermissionDenied, "scope not allowed", "", http.StatusForbidden)
				return
			}

			for _, pattern := range allowed {
				if matchResource(pattern, resource) {
					next.ServeHTTP(w, r)
					return
				}
			}

			api.WriteError(w, api.ErrPermissionDenied, "resource not allowed", "", http.StatusForbidden)
		})
	}
}

func matchResource(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == resource {
		return true
	}
	return false
}

func RequireScope(scopes ...auth.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			keyScopes := auth.ScopesFromContext(ctx)
			role := auth.RoleFromContext(ctx)

			if role == auth.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			hasRequired := false
			for _, required := range scopes {
				for _, s := range keyScopes {
					if s == required {
						hasRequired = true
						break
					}
				}
				if hasRequired {
					break
				}
			}

			if !hasRequired {
				api.WriteError(w, api.ErrPermissionDenied, "insufficient permissions", "", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(roles ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			role := auth.RoleFromContext(ctx)

			for _, allowed := range roles {
				if role == allowed || role == auth.RoleAdmin {
					next.ServeHTTP(w, r)
					return
				}
			}

			api.WriteError(w, api.ErrPermissionDenied, "role not authorized", "", http.StatusForbidden)
		})
	}
}
