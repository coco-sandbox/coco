# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

.PHONY: all build-go build-zig generate-ebpf test test-go test-zig clean proto

GOFLAGS := -trimpath

all: build-go build-zig

build-go: bin/coco-gateway bin/coco-master bin/coco-node bin/coco-checkpoint bin/coco-net bin/cococtl

bin/coco-gateway: $(shell find cmd/coco-gateway pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-gateway

bin/coco-master: $(shell find cmd/coco-master pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-master

bin/coco-node: $(shell find cmd/coco-node pkg -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/coco-node

bin/coco-checkpoint: $(shell find daemon/coco-checkpoint -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./daemon/coco-checkpoint/cmd

bin/coco-net: generate-ebpf $(shell find daemon/coco-net -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./daemon/coco-net/cmd

bin/cococtl: $(shell find cmd/cococtl -name '*.go' 2>/dev/null)
	mkdir -p bin
	go build $(GOFLAGS) -o $@ ./cmd/cococtl

build-zig:
	cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe
	cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe
	cd daemon/coco-fork && zig build -Doptimize=ReleaseSafe

generate-ebpf:
	cd daemon/coco-net/ebpf && go generate

test-go:
	go test ./pkg/... ./cmd/...

test-zig:
	cd daemon/coco-visor && zig build test
	cd daemon/coco-agent && zig build test

test: test-go test-zig

proto:
	rm -rf /tmp/coco-proto
	mkdir -p /tmp/coco-proto/v1 /tmp/coco-proto/v1connect /tmp/coco-proto/internal
	PATH="$(HOME)/go/bin:$$PATH" protoc \
		--go_out=/tmp/coco-proto \
		--go_opt=Mproto/coco/v1/coco.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/coco/v1/checkpoint.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/coco/v1/node.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/coco/v1/master.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/coco/v1/cluster.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/coco/v1/network.proto=coco/pkg/api/v1 \
		--go_opt=Mproto/internal/visor.proto=coco/pkg/api/internal \
		--go_opt=Mproto/internal/agent.proto=coco/pkg/api/internal \
		-I. -Iproto \
		proto/coco/v1/coco.proto \
		proto/coco/v1/checkpoint.proto \
		proto/coco/v1/node.proto \
		proto/coco/v1/master.proto \
		proto/coco/v1/cluster.proto \
		proto/coco/v1/network.proto \
		proto/internal/visor.proto \
		proto/internal/agent.proto
	PATH="$(HOME)/go/bin:$$PATH" protoc \
		--connect-go_out=/tmp/coco-proto \
		--connect-go_opt=Mproto/coco/v1/coco.proto=coco/pkg/api/v1 \
		--connect-go_opt=Mproto/coco/v1/checkpoint.proto=coco/pkg/api/v1 \
		--connect-go_opt=Mproto/coco/v1/node.proto=coco/pkg/api/v1 \
		--connect-go_opt=Mproto/coco/v1/master.proto=coco/pkg/api/v1 \
		--connect-go_opt=Mproto/coco/v1/cluster.proto=coco/pkg/api/v1 \
		--connect-go_opt=Mproto/coco/v1/network.proto=coco/pkg/api/v1 \
		-I. -Iproto \
		proto/coco/v1/coco.proto \
		proto/coco/v1/checkpoint.proto \
		proto/coco/v1/node.proto \
		proto/coco/v1/master.proto \
		proto/coco/v1/cluster.proto \
		proto/coco/v1/network.proto
	rm -rf pkg/api/v1 pkg/api/v1connect pkg/api/internal
	cp -r /tmp/coco-proto/v1 pkg/api/
	cp -r /tmp/coco-proto/v1connect pkg/api/
	cp -r /tmp/coco-proto/internal pkg/api/
	rm -rf /tmp/coco-proto

clean:
	rm -rf bin/
	rm -f daemon/coco-visor/zig-out/bin/*
	rm -f daemon/coco-agent/zig-out/bin/*
	rm -f daemon/coco-fork/zig-out/bin/*
	rm -f daemon/coco-net/ebpf/*_bpf*.go daemon/coco-net/ebpf/*_bpf*.o
