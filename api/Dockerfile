# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.25.12-alpine AS builder

LABEL maintainer="Luas Team <team@luas.dev>"

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/ ./cmd/server ./cmd/luas

# Run stage — distroless static for minimal image size and fast pull
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/server /app/luas-server
COPY --from=builder /out/luas /app/luas

ENV TZ=UTC \
    APP_ENV=production \
    APP_DEBUG=false \
    SERVER_MODE=release \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    LOG_STDOUT=true \
    LOG_FILE_ENABLED=false \
    LOG_JSON=true

EXPOSE 8025

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/luas", "health:check", "--timeout=2s"]

ENTRYPOINT ["/app/luas-server"]
