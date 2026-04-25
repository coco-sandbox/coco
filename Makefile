# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

.PHONY: all build test clean proto go-zig go-core cocovisor coconet cocofork

# Go components
go-core:
	cd core && go build -o coco-core .

go-ctl:
	cd ctl && go build -o cococtl .

# Zig components
cocovisor:
	cd src/cocovisor && zig build -Drelease-safe=true

coconet:
	cd src/coconet && zig build -Drelease-safe=true

cocofork:
	cd src/cocofork && zig build -Drelease-safe=true

cocod:
	cd src/cocod && zig build -Drelease-safe=true

# Build all
all: go-core go-ctl cocovisor coconet cocofork cocod

# Test
test-go:
	cd core && go test ./...
	cd ctl && go test ./...

test-zig:
	cd src/cocovisor && zig build test
	cd src/coconet && zig build test
	cd src/cocofork && zig build test

test: test-go test-zig

# Protobuf generation
proto:
	cd proto && make gen-go

# Clean
clean:
	rm -f core/coco-core ctl/cococtl
	rm -f src/cocovisor/zig-out/bin/cocovisor
	rm -f src/coconet/zig-out/bin/coconet
	rm -f src/cocofork/zig-out/bin/cocofork
	rm -f src/cocod/zig-out/bin/cocod