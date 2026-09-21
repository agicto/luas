# syntax=docker.io/docker/dockerfile:1.24.0@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89
ARG OCI_SOURCE=https://github.com/zgiai/luas
ARG OCI_REVISION=unknown
ARG OCI_VERSION=dev

# Build stage
FROM docker.io/library/golang:1.27.1-alpine3.24@sha256:4cb7ac979db5fcc41cae44b2227ba5ab8a51e8807f40d9ba4dee20a0ad960b5b AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/ ./cmd/server ./cmd/luas

# Run stage - distroless static for a shell-free, non-root runtime.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b

ARG OCI_SOURCE
ARG OCI_REVISION
ARG OCI_VERSION

LABEL org.opencontainers.image.title="Luas API" \
    org.opencontainers.image.description="Production runtime for the Luas Go API scaffold" \
    org.opencontainers.image.source="${OCI_SOURCE}" \
    org.opencontainers.image.revision="${OCI_REVISION}" \
    org.opencontainers.image.version="${OCI_VERSION}" \
    org.opencontainers.image.base.name="gcr.io/distroless/static-debian12:nonroot" \
    org.opencontainers.image.base.digest="sha256:aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b"

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
