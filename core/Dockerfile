# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY core/go.mod core/go.sum ./
RUN go mod download

# Copy source
COPY core/ ./core/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o coco-core ./core/

# Runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Install Cloud Hypervisor
RUN apk add --no-cache curl && \
    curl -L https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v36.0/clh -o /usr/local/bin/clh && \
    chmod +x /usr/local/bin/clh

# Install iproute2 (for network setup)
RUN apk add --no-cache iproute2 iptables

# Create directories
RUN mkdir -p /run/coco /var/lib/coco/{images,hibernation,checkpoints,replays,store}

COPY --from=builder /build/coco-core /usr/local/bin/

EXPOSE 4747 4748 9090

ENV COCO_RUN_DIR=/run/coco
ENV COCO_DATA_DIR=/var/lib/coco
ENV COCO_LOG_LEVEL=info

ENTRYPOINT ["coco-core"]
CMD ["--help"]