# Multi-stage build for Hyperion Coordinator MCP Server
# Produces a minimal Alpine-based image with the Go binary

# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o hyperion-coordinator-mcp \
    main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 hyperion && \
    adduser -D -u 1000 -G hyperion hyperion

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/hyperion-coordinator-mcp .

# Copy any additional files needed
COPY --from=builder /build/README.md .

# Change ownership to non-root user
RUN chown -R hyperion:hyperion /app

# Switch to non-root user
USER hyperion

# Expose health check endpoint (if added in future)
# EXPOSE 8080

# Set default environment variables
ENV MONGODB_URI="" \
    MONGODB_DATABASE="coordinator_db" \
    LOG_LEVEL="info"

# Health check (optional - requires HTTP endpoint)
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD ["/app/hyperion-coordinator-mcp", "health"]

# Run the MCP server
ENTRYPOINT ["/app/hyperion-coordinator-mcp"]
