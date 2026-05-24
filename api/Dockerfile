# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.24-alpine AS builder

LABEL maintainer="ZGO Team <team@eogo-dev.com>"

WORKDIR /app

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/ ./cmd/server ./cmd/zgo

# Run stage — distroless static for minimal image size and fast pull
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/server /app/zgo-server
COPY --from=builder /out/zgo /app/zgo
COPY --from=builder /app/.env.example /app/.env

ENV TZ=Asia/Shanghai

EXPOSE 8025

ENTRYPOINT ["/app/zgo-server"]
