# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

.PHONY: test integration-test clean

# Run unit tests
test:
	go test -v ./...

# Run integration tests (requires coco-core running)
integration-test:
	go test -v -tags=integration ./tests/integration/...

# Run all tests including integration
all-tests: test integration-test

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Clean test artifacts
clean:
	go clean -testcache

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
race:
	go test -race -v ./...
