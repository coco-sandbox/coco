# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

.PHONY: all build-go build-zig test test-go test-zig clean proto

GOFLAGS := -trimpath

all: build-go build-zig

build-go: bin/coco-gateway bin/coco-master bin/coco-node bin/cococtl

bin/coco-gateway: $(shell find cmd/coco-gateway pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-gateway

bin/coco-master: $(shell find cmd/coco-master pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-master

bin/coco-node: $(shell find cmd/coco-node pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-node

bin/cococtl: $(shell find cmd/cococtl -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/cococtl

build-zig:
	cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe
	cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe
	cd daemon/coco-fork && zig build -Doptimize=ReleaseSafe
	cd daemon/coco-net && zig build -Doptimize=ReleaseSafe

test-go:
	go test ./pkg/... ./cmd/...

test-zig:
	cd daemon/coco-visor && zig build test
	cd daemon/coco-agent && zig build test

test: test-go test-zig

proto:
	protoc --go_out=. --go-grpc_out=. proto/coco/v1/*.proto

clean:
	rm -rf bin/
	rm -f daemon/coco-visor/zig-out/bin/*
	rm -f daemon/coco-agent/zig-out/bin/*
	rm -f daemon/coco-fork/zig-out/bin/*
	rm -f daemon/coco-net/zig-out/bin/*
