#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Run all tests
# Usage: ./scripts/test/test-all.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Running all tests ==="

cd "$ROOT_DIR"

echo "Running Go tests..."
make test-go

echo "Running Zig tests..."
make test-zig

echo "=== All tests completed ==="
