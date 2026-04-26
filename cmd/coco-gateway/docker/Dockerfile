FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /coco-gateway ./cmd/coco-gateway

FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /coco-gateway /app/
COPY configs/ /app/configs/

ENV COCO_LISTEN_ADDR=:4747
ENV COCO_DATA_DIR=/data

EXPOSE 4747 8080

ENTRYPOINT ["/app/coco-gateway"]
