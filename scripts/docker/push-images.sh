#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Push container images to registry
# Usage: ./scripts/docker/push-images.sh [registry] [tag]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

REGISTRY="${1:-ghcr.io/$USER}"
TAG="${2:-latest}"

echo "=== Pushing images to $REGISTRY (tag: $TAG) ==="

for name in coco-gateway coco-master coco-node coco-visor coco-agent cococtl; do
    echo "Pushing $name..."
    docker tag "$name:$TAG" "$REGISTRY/$name:$TAG"
    docker push "$REGISTRY/$name:$TAG"
done

echo "=== All images pushed ==="
