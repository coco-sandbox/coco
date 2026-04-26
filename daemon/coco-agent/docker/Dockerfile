FROM ghcr.io/ziglang/rig:0.14 AS builder

WORKDIR /build

COPY . .

RUN cd daemon/coco-agent && zig build -Doptimize=ReleaseSafe

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/daemon/coco-agent/zig-out/bin/* /app/

ENTRYPOINT ["/app/coco-agent"]
