# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS builder
WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/wolt ./cmd/wolt && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/wolt-mcp ./cmd/wolt-mcp

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app
COPY --from=builder /out/wolt /usr/local/bin/wolt
COPY --from=builder /out/wolt-mcp /usr/local/bin/wolt-mcp
USER app
ENV HOME=/home/app
WORKDIR /home/app
EXPOSE 8080
ENTRYPOINT ["wolt-mcp"]
