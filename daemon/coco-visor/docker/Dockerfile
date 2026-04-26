FROM ghcr.io/ziglang/rig:0.14 AS builder

WORKDIR /build

COPY . .

RUN cd daemon/coco-visor && zig build -Doptimize=ReleaseSafe

FROM alpine:3.19

RUN apk add --no-cache ca-certificates kvm-host

WORKDIR /app

COPY --from=builder /build/daemon/coco-visor/zig-out/bin/* /app/

RUN mkdir -p /run/coco

ENV COCO_VISOR_SOCKET=/run/coco/visor.sock

ENTRYPOINT ["/app/coco-visor"]
