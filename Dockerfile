# Multi-stage Dockerfile for Event Trigger Platform
# Optimized for production with minimal image size and security best practices

# Stage 1: Builder
FROM golang:1.22.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
# Using go.* pattern to copy go.mod and go.sum (if exists)
COPY go.* ./

# Download dependencies (cached if go.mod unchanged)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build all services with optimized flags
# -trimpath: removes file system paths from compiled executable
# -ldflags="-s -w": strips debug info (-s) and DWARF symbol table (-w)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN go build -trimpath -ldflags="-s -w" -o /build/bin/api ./cmd/api && \
    go build -trimpath -ldflags="-s -w" -o /build/bin/scheduler ./cmd/scheduler

# Stage 2: Runtime - API Server
FROM alpine:3.19 AS api

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/api /app/api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Use non-root user
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]

# Stage 3: Runtime - Scheduler
FROM alpine:3.19 AS scheduler

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /build/bin/scheduler /app/scheduler
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER appuser

ENTRYPOINT ["/app/scheduler"]

