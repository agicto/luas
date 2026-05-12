# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.24-alpine AS builder

LABEL maintainer="ZGO Team <team@eogo-dev.com>"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o zgo-server cmd/server/main.go
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o zgo cmd/zgo/main.go

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /app/config /app/logs /app/storage

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/zgo-server .
COPY --from=builder /app/zgo .
COPY --from=builder /app/.env.example ./.env

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8025

HEALTHCHECK NONE

CMD ["./zgo-server"]
