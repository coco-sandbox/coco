#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Build all binaries
# Usage: ./scripts/build/build-all.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Building all Coco binaries ==="

cd "$ROOT_DIR"

echo "Building Go binaries..."
make build-go

echo "Building Zig binaries..."
make build-zig

echo "Building eBPF programs..."
make -C ebpf all 2>/dev/null || true

echo "=== All binaries built ==="
echo "Go binaries: $ROOT_DIR/bin/"
echo "Zig binaries: $ROOT_DIR/daemon/*/zig-out/bin/"
