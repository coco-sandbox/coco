#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Generate Go code from protobuf definitions
# Usage: ./scripts/tools/generate-proto.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROTO_DIR="$ROOT_DIR/proto"
OUT_DIR="$ROOT_DIR/pkg"

echo "=== Coco Proto Generator ==="
echo "Proto dir: $PROTO_DIR"
echo "Output dir: $OUT_DIR"

# Check prerequisites
command -v protoc >/dev/null 2>&1 || { echo "protoc not found"; exit 1; }
command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go not found"; exit 1; }
command -v protoc-gen-connect-go >/dev/null 2>&1 || { echo "protoc-gen-connect-go not found"; exit 1; }

# Clean output directory
rm -rf "$OUT_DIR/github.com"
mkdir -p "$OUT_DIR"

# Generate with explicit module mapping to pkg/api/v1
echo "Generating Go + Connect code..."
protoc \
    --go_out="$OUT_DIR" \
    --go_opt=Mproto/coco/v1/coco.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=Mproto/coco/v1/checkpoint.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=Mproto/coco/v1/node.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=Mproto/coco/v1/master.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=Mproto/coco/v1/cluster.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=Mproto/coco/v1/network.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --go_opt=module=github.com/coco-sandbox/coco \
    --connect-go_out="$OUT_DIR" \
    --connect-go_opt=Mproto/coco/v1/coco.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=Mproto/coco/v1/checkpoint.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=Mproto/coco/v1/node.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=Mproto/coco/v1/master.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=Mproto/coco/v1/cluster.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=Mproto/coco/v1/network.proto=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_opt=module=github.com/coco-sandbox/coco \
    -I"$ROOT_DIR" \
    -I"$PROTO_DIR" \
    "$PROTO_DIR"/coco/v1/*.proto

# Move files to correct location
mkdir -p "$OUT_DIR/api/v1"
mv "$OUT_DIR/github.com/coco-sandbox/coco/pkg/api/v1/"* "$OUT_DIR/api/v1/" 2>/dev/null || true
rm -rf "$OUT_DIR/github.com"

echo "=== Generated files ==="
find "$OUT_DIR" -type f -name "*.go" | sort
echo ""
echo "✅ Proto generation complete"
