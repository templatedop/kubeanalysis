# Build stage
FROM golang:1.21-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildTime=${BUILD_TIME}" \
    -o /kubeanalysis \
    ./cmd/kubeanalysis

# Runtime stage
FROM alpine:3.19

# Labels
LABEL org.opencontainers.image.title="KubeAnalysis" \
      org.opencontainers.image.description="Kubernetes cluster analysis with RAG and Temporal" \
      org.opencontainers.image.vendor="KubeAnalysis" \
      org.opencontainers.image.source="https://github.com/kubeanalysis/kubeanalysis"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 kubeanalysis && \
    adduser -u 1000 -G kubeanalysis -s /bin/sh -D kubeanalysis

# Create directories
RUN mkdir -p /app/knowledge /app/config && \
    chown -R kubeanalysis:kubeanalysis /app

# Copy binary from builder
COPY --from=builder /kubeanalysis /usr/local/bin/kubeanalysis

# Copy knowledge base
COPY --from=builder /app/knowledge /app/knowledge

# Set ownership
RUN chown -R kubeanalysis:kubeanalysis /app

# Switch to non-root user
USER kubeanalysis

# Set working directory
WORKDIR /app

# Expose ports
# 8080 - API/Health endpoints
# 9090 - Metrics (optional separate port)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/live || exit 1

# Default entrypoint
ENTRYPOINT ["kubeanalysis"]

# Default command (can be overridden)
CMD ["worker"]
