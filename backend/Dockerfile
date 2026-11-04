# ==============================================================================
# PUJA VAAHINI BACKEND GATEWAY DOCKERFILE
# Multi-stage production container build (Go 1.22+ Alpine -> Minimal Scratch/Alpine)
# ==============================================================================

# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile optimized static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/api-gateway \
    ./cmd/api-gateway/main.go

# Production Runtime Stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root system user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/api-gateway /app/api-gateway

USER appuser

EXPOSE 8085

ENV PORT=8085 \
    GIN_MODE=release \
    ENVIRONMENT=production

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8085/health || exit 1

ENTRYPOINT ["/app/api-gateway"]
