FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /coco-master ./cmd/coco-master

FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /coco-master /app/
COPY configs/ /app/configs/

ENV COCO_GRPC_ADDR=:4746
ENV COCO_DATA_DIR=/data

EXPOSE 4746

ENTRYPOINT ["/app/coco-master"]
