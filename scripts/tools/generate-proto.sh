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
if ! command -v protoc >/dev/null 2>&1; then
    echo "protoc not found"
    exit 1
fi

# Add Go bin to PATH
export PATH="$PATH:$(go env GOPATH)/bin"

if ! command -v protoc-gen-go >/dev/null 2>&1; then
    echo "protoc-gen-go not found in PATH"
    exit 1
fi

if ! command -v protoc-gen-connect-go >/dev/null 2>&1; then
    echo "protoc-gen-connect-go not found in PATH"
    exit 1
fi

# Clean output directories
rm -rf "$OUT_DIR/github.com" "$OUT_DIR/pkg"
rm -rf "$OUT_DIR/api/v1" "$OUT_DIR/api/v1connect" "$OUT_DIR/api/internal"
mkdir -p "$OUT_DIR/api/v1"
mkdir -p "$OUT_DIR/api/v1connect"
mkdir -p "$OUT_DIR/api/internal"

cd "$ROOT_DIR"

# Generate public API (proto/coco/v1/)
echo "Generating public API..."
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

# Generate internal API (proto/internal/)
echo "Generating internal API..."
protoc \
    --go_out="$OUT_DIR" \
    --go_opt=Mproto/internal/visor.proto=github.com/coco-sandbox/coco/pkg/api/internal \
    --go_opt=Mproto/internal/agent.proto=github.com/coco-sandbox/coco/pkg/api/internal \
    --go_opt=module=github.com/coco-sandbox/coco \
    -I"$ROOT_DIR" \
    -I"$PROTO_DIR" \
    "$PROTO_DIR"/internal/*.proto

# Move files from nested pkg/pkg/ to correct location
if [ -d "$OUT_DIR/pkg/api/v1" ]; then
    cp -r "$OUT_DIR/pkg/api/v1/"* "$OUT_DIR/api/v1/"
    rm -rf "$OUT_DIR/pkg"
fi

# Move connect files to correct location
if [ -d "$OUT_DIR/pkg/api/v1/v1connect" ]; then
    cp -r "$OUT_DIR/pkg/api/v1/v1connect/"* "$OUT_DIR/api/v1connect/"
fi

# Also check direct output location
if [ -d "$OUT_DIR/github.com/coco-sandbox/coco/pkg/api/v1" ]; then
    cp -r "$OUT_DIR/github.com/coco-sandbox/coco/pkg/api/v1/"* "$OUT_DIR/api/v1/"
    rm -rf "$OUT_DIR/github.com"
fi

# Move internal files
if [ -d "$OUT_DIR/github.com/coco-sandbox/coco/pkg/api/internal" ]; then
    cp -r "$OUT_DIR/github.com/coco-sandbox/coco/pkg/api/internal/"* "$OUT_DIR/api/internal/"
fi

echo "=== Generated files ==="
echo "Public API:"
find "$OUT_DIR/api/v1" -type f -name "*.go" 2>/dev/null | sort
find "$OUT_DIR/api/v1connect" -type f -name "*.go" 2>/dev/null | sort
echo ""
echo "Internal API:"
find "$OUT_DIR/api/internal" -type f -name "*.go" 2>/dev/null | sort
echo ""
echo "✅ Proto generation complete"
