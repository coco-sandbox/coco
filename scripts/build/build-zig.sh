#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Build Zig binaries
# Usage: ./scripts/build/build-zig.sh [debug|release-safe|release-small]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

BUILD_MODE="${1:-ReleaseSafe}"

echo "=== Building Zig binaries (mode: $BUILD_MODE) ==="

case "$BUILD_MODE" in
    debug)
        ZIG_MODE="-Ddebug"
        ;;
    release-safe)
        ZIG_MODE="-Doptimize=ReleaseSafe"
        ;;
    release-small)
        ZIG_MODE="-Doptimize=ReleaseSmall"
        ;;
    *)
        echo "Unknown build mode: $BUILD_MODE"
        echo "Usage: $0 [debug|release-safe|release-small]"
        exit 1
        ;;
esac

cd "$ROOT_DIR/daemon/coco-visor"
echo "Building coco-visor..."
zig build $ZIG_MODE

cd "$ROOT_DIR/daemon/coco-agent"
echo "Building coco-agent..."
zig build $ZIG_MODE

cd "$ROOT_DIR/daemon/coco-fork"
echo "Building coco-fork..."
zig build $ZIG_MODE

echo "=== Built Zig binaries ==="
ls -la "$ROOT_DIR/daemon/coco-visor/zig-out/bin/" 2>/dev/null || true
ls -la "$ROOT_DIR/daemon/coco-agent/zig-out/bin/" 2>/dev/null || true
ls -la "$ROOT_DIR/daemon/coco-fork/zig-out/bin/" 2>/dev/null || true
