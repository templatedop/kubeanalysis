# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo '0.1.0')" \
    -o /app/bin/kubeanalysis \
    ./cmd/kubeanalysis

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' kubeanalysis

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/kubeanalysis /app/kubeanalysis

# Copy default config if exists
COPY --from=builder /app/config.yaml* /app/

# Set ownership
RUN chown -R kubeanalysis:kubeanalysis /app

# Switch to non-root user
USER kubeanalysis

# Expose metrics port (if applicable)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/kubeanalysis"]

# Default command
CMD ["worker"]
