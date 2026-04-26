#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Build Go binaries
# Usage: ./scripts/build/build-go.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="$ROOT_DIR/bin"

echo "=== Building Go binaries ==="

cd "$ROOT_DIR"

mkdir -p "$OUT_DIR"

echo "Building coco-gateway..."
go build -trimpath -o "$OUT_DIR/coco-gateway" ./cmd/coco-gateway

echo "Building coco-master..."
go build -trimpath -o "$OUT_DIR/coco-master" ./cmd/coco-master

echo "Building coco-node..."
go build -trimpath -o "$OUT_DIR/coco-node" ./cmd/coco-node

echo "Building cococtl..."
go build -trimpath -o "$OUT_DIR/cococtl" ./cmd/cococtl

echo "=== Built binaries ==="
ls -la "$OUT_DIR"
