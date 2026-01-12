# Build stage
FROM golang:1.24-alpine AS builder

# Set author label
LABEL maintainer="ZGO Team <team@eogo-dev.com>"

# Install git for dependency downloading
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zgo-server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zgo cmd/zgo/main.go

# Run stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Set timezone
ENV TZ=Asia/Shanghai

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Create necessary directories
RUN mkdir -p /app/config /app/logs /app/storage
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/zgo-server .
COPY --from=builder /app/zgo .
COPY --from=builder /app/.env.example ./.env

# Set permissions
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8025

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8025/health || exit 1

# Start command
CMD ["./zgo-server"]
