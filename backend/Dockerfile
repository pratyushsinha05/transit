# ============================================================================
# Stage 1: Builder
# ============================================================================
FROM golang:alpine AS builder

# Install build dependencies (required for h3-go CGO)
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /app

# Build arguments for version info
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_HASH=unknown

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
# CGO_ENABLED=1 required for h3-go
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.GitHash=${GIT_HASH}" \
    -o server \
    ./cmd/server

# ============================================================================
# Stage 2: Runtime
# ============================================================================
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    # Required for CGO binaries
    libc6-compat

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy migrations (for container-based migration if needed)
COPY --from=builder /app/migrations /app/migrations

# Copy public assets if they exist
# COPY --from=builder /app/public /app/public

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./server"]
