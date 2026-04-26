FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /coco-node ./cmd/coco-node

FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl kvm-host

WORKDIR /app

COPY --from=builder /coco-node /app/
COPY configs/ /app/configs/

ENV COCO_GRPC_ADDR=:4748
ENV COCO_DATA_DIR=/data
ENV COCO_POOL_SIZE=10

EXPOSE 4748 4747

ENTRYPOINT ["/app/coco-node"]
