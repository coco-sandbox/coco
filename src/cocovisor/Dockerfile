# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

FROM alpine:3.19 AS builder

# Install Zig toolchain
RUN apk add --no-cache curl xz bash
RUN curl -L https://ziglang.org/download/0.14.0/zig-linux-x86_64-0.14.0.tar.xz -o /tmp/zig.tar.xz && \
    tar -xf /tmp/zig.tar.xz -C /usr/local --strip-components=1 && \
    rm /tmp/zig.tar.xz

WORKDIR /build

# Copy source
COPY src/cocovisor/ ./src/cocovisor/

# Build release
RUN cd src/cocovisor && zig build -Drelease-safe=true -p /build/out

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/out/bin/cocovisor /usr/local/bin/

ENTRYPOINT ["cocovisor"]