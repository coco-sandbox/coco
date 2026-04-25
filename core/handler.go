// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"io"
	"net/http"
)

// =============================================================================
// Handler Helpers
// =============================================================================

func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}