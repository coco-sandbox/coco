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
OUT_DIR="$ROOT_DIR/pkg/api/v1"

echo "=== Coco Proto Generator ==="
echo "Proto dir: $PROTO_DIR"
echo "Output dir: $OUT_DIR"

# Check prerequisites
command -v protoc >/dev/null 2>&1 || { echo "protoc not found"; exit 1; }
command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go not found"; exit 1; }
command -v protoc-gen-connect-go >/dev/null 2>&1 || { echo "protoc-gen-connect-go not found"; exit 1; }

# Clean output directory
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

# Generate
echo "Generating Go + Connect code..."
protoc \
    --go_out="$OUT_DIR" \
    --go_opt=Mproto/coco/v1/coco.proto=github.com/coco-sandbox/coco/pkg/api/v1/coco \
    --go_opt=Mproto/coco/v1/checkpoint.proto=github.com/coco-sandbox/coco/pkg/api/v1/checkpoint \
    --go_opt=Mproto/coco/v1/node.proto=github.com/coco-sandbox/coco/pkg/api/v1/node \
    --go_opt=Mproto/coco/v1/master.proto=github.com/coco-sandbox/coco/pkg/api/v1/master \
    --go_opt=module=github.com/coco-sandbox/coco/pkg/api/v1 \
    --connect-go_out="$OUT_DIR" \
    --connect-go_opt=Mproto/coco/v1/coco.proto=github.com/coco-sandbox/coco/pkg/api/v1/coco \
    --connect-go_opt=Mproto/coco/v1/checkpoint.proto=github.com/coco-sandbox/coco/pkg/api/v1/checkpoint \
    --connect-go_opt=Mproto/coco/v1/node.proto=github.com/coco-sandbox/coco/pkg/api/v1/node \
    --connect-go_opt=Mproto/coco/v1/master.proto=github.com/coco-sandbox/coco/pkg/api/v1/master \
    --connect-go_opt=module=github.com/coco-sandbox/coco/pkg/api/v1 \
    -I"$PROTO_DIR" \
    "$PROTO_DIR"/coco/v1/*.proto

echo "=== Generated files ==="
find "$OUT_DIR" -type f | sort
echo ""
echo "✅ Proto generation complete"
