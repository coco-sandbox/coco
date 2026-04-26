#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Run integration tests
# Usage: ./scripts/test/test-integration.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Running integration tests ==="

cd "$ROOT_DIR"

echo "Starting docker-compose..."
docker-compose up -d

echo "Waiting for services to be ready..."
for i in $(seq 1 60); do
    if curl -sf http://localhost:4747/health | grep -q '"healthy":true'; then
        echo "Services are ready"
        break
    fi
    echo "Waiting for services... ($i/60)"
    sleep 2
done

echo "Running integration tests..."
go test -v ./test/integration/...

echo "Cleaning up..."
docker-compose down -v

echo "=== Integration tests completed ==="
