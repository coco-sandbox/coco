#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Format code for all languages
# Usage: ./scripts/tools/format-code.sh [--check]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

CHECK_MODE=false
if [[ "$1" == "--check" ]]; then
    CHECK_MODE=true
fi

echo "=== Coco Code Formatter ==="
cd "$ROOT_DIR"

# Format Go code
format_go() {
    echo "Formatting Go code..."
    if [[ "$CHECK_MODE" == "true" ]]; then
        gofmt -l -s cmd/ pkg/ || true
    else
        gofmt -l -s -w cmd/ pkg/
        go fmt ./cmd/ ./pkg/
    fi
}

# Format Zig code
format_zig() {
    echo "Formatting Zig code..."
    if command -v zig >/dev/null 2>&1; then
        if [[ "$CHECK_MODE" == "true" ]]; then
            find daemon/ -name "*.zig" -exec zig fmt --check {} \; || true
        else
            find daemon/ -name "*.zig" -exec zig fmt -w {} \; 2>/dev/null || true
        fi
    else
        echo "  zig not found, skipping Zig formatting"
    fi
}

# Format Shell scripts
format_shell() {
    echo "Formatting shell scripts..."
    if command -v shfmt >/dev/null 2>&1; then
        if [[ "$CHECK_MODE" == "true" ]]; then
            shfmt -l scripts/ || true
        else
            shfmt -l -w scripts/
        fi
    else
        echo "  shfmt not found, skipping shell formatting"
    fi
}

# Format C/eBPF code
format_c() {
    echo "Formatting C/eBPF code..."
    if command -v clang-format >/dev/null 2>&1; then
        if [[ "$CHECK_MODE" == "true" ]]; then
            find ebpf/ -name "*.c" -o -name "*.h" | xargs clang-format --dry-run --Werror 2>/dev/null || true
        else
            find ebpf/ -name "*.c" -o -name "*.h" | xargs clang-format -i 2>/dev/null || true
        fi
    else
        echo "  clang-format not found, skipping C formatting"
    fi
}

format_go
format_zig
format_shell
format_c

if [[ "$CHECK_MODE" == "true" ]]; then
    echo "Format check complete."
else
    echo "Code formatting complete."
fi