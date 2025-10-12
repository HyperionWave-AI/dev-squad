# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
COPY mcp-server/go.mod mcp-server/go.sum ./mcp-server/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the unified coordinator binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o coordinator ./cmd/coordinator

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/coordinator .

# Copy UI dist files (for UI serving)
COPY ui/dist ./ui/dist

# Expose ports
# 7095: HTTP REST API + UI serving
# 7096: MCP stdio (not exposed, used internally)
EXPOSE 7095

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=10s \
  CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:7095/health || exit 1

# Run coordinator in HTTP-only mode by default
CMD ["./coordinator", "-mode", "http"]
