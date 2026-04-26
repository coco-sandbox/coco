#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Build container images
# Usage: ./scripts/docker/build-images.sh [tag]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

TAG="${1:-latest}"

echo "=== Building container images (tag: $TAG) ==="

cd "$ROOT_DIR"

build_image() {
    local name="$1"
    local context="$2"
    local dockerfile="$3"

    echo "Building $name..."
    docker build -t "coco-$name:$TAG" -f "$dockerfile" "$context"
}

build_image "coco-gateway" "." "cmd/coco-gateway/docker/Dockerfile"
build_image "coco-master" "." "cmd/coco-master/docker/Dockerfile"
build_image "coco-node" "." "cmd/coco-node/docker/Dockerfile"
build_image "coco-visor" "daemon/coco-visor" "daemon/coco-visor/docker/Dockerfile"
build_image "coco-agent" "daemon/coco-agent" "daemon/coco-agent/docker/Dockerfile"

echo "=== Built images ==="
docker images | grep coco-
