# Build stage
FROM golang:1.24-alpine AS builder

LABEL maintainer="ZGO Team <team@eogo-dev.com>"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o zgo-server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o zgo cmd/zgo/main.go

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl

ENV TZ=Asia/Shanghai

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN mkdir -p /app/config /app/logs /app/storage

WORKDIR /app

COPY --from=builder /app/zgo-server .
COPY --from=builder /app/zgo .
COPY --from=builder /app/.env.example ./.env

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8025

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8025/health || exit 1

CMD ["./zgo-server"]
