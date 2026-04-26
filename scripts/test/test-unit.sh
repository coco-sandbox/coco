#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Run unit tests
# Usage: ./scripts/test/test-unit.sh [package]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

PACKAGE="${1:-./pkg/... ./cmd/...}"

echo "=== Running unit tests ==="

cd "$ROOT_DIR"

go test -v -race $PACKAGE

echo "=== Unit tests completed ==="
