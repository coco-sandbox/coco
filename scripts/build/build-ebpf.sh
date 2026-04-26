#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# Build eBPF programs
# Usage: ./scripts/build/build-ebpf.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
EBPF_DIR="$ROOT_DIR/ebpf"

echo "=== Building eBPF programs ==="

command -v clang >/dev/null 2>&1 || { echo "clang not found"; exit 1; }
command -v llc >/dev/null 2>&1 || { echo "llc not found"; exit 1; }

cd "$EBPF_DIR"

compile_bpf() {
    local name="$1"
    local src="$name.bpf.c"
    local out="$name.o"

    if [ -f "$src" ]; then
        echo "Compiling $src..."
        clang -O2 -target bpf -c "$src" -o "$out" -Iheaders
    fi
}

for dir in from_sandbox from_host from_world xdp shaper; do
    if [ -d "$dir" ]; then
        echo "Processing $dir..."
        for f in "$dir"/*.bpf.c; do
            if [ -f "$f" ]; then
                name=$(basename "$f" .bpf.c)
                compile_bpf "$dir/$name"
            fi
        done
    fi
done

echo "=== Built eBPF objects ==="
find "$EBPF_DIR" -name "*.o" -type f
