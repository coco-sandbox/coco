FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /coco-net ./daemon/coco-net

FROM alpine:3.19

RUN apk add --no-cache ca-certificates iproute2

WORKDIR /app

COPY --from=builder /coco-net /app/

RUN mkdir -p /run/coco

ENV COCO_NET_SOCKET=/run/coco/net.sock

ENTRYPOINT ["/app/coco-net"]
