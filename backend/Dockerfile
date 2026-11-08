# ==============================================================================
# PUJA VAAHINI BACKEND GATEWAY DOCKERFILE
# Multi-stage production container build (Go Alpine -> Minimal Alpine Runtime)
# ==============================================================================

# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile optimized static binaries for both API gateway and migration runner
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/api-gateway \
    ./cmd/api-gateway

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/migrate \
    ./cmd/migrate

# Production Runtime Stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root system user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled binaries and schema migrations into runtime image
COPY --from=builder /app/bin/api-gateway /app/api-gateway
COPY --from=builder /app/bin/migrate /app/migrate
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/seeds /app/seeds

# Set permissions for non-root runtime
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8085

ENV PORT=8085 \
    GIN_MODE=release \
    ENVIRONMENT=production \
    MIGRATIONS_DIR=/app/migrations

# Healthcheck uses dynamic Railway PORT environment variable with fallback to 8085
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD sh -c 'curl -f http://127.0.0.1:${PORT:-8085}/health || exit 1'

CMD ["/bin/sh", "-c", "/app/migrate up && exec /app/api-gateway"]
