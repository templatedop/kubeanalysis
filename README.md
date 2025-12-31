# KubeAnalysis

Intelligent Kubernetes cluster analysis powered by RAG (Retrieval-Augmented Generation) and Temporal workflows.

## Features

- **Security Analysis**: Detect misconfigurations, vulnerabilities, and compliance violations
- **Performance Analysis**: Identify resource bottlenecks, inefficiencies, and optimization opportunities
- **Predictive Analysis**: Anticipate potential issues using LLM-powered insights
- **RAG-Powered Knowledge**: Leverage best practices from NSA/CISA, CIS benchmarks, and custom knowledge bases
- **Temporal Workflows**: Reliable, scalable analysis orchestration with durable execution
- **Production Ready**: Health endpoints, structured logging, metrics, rate limiting, and graceful shutdown

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         KubeAnalysis                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Temporal   │  │     RAG     │  │    Observability       │  │
│  │  Workflows  │  │   Engine    │  │  (Logs/Metrics/Health) │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│         │               │                     │                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Analyzers                             │   │
│  │  ┌──────────┐  ┌─────────────┐  ┌────────────────────┐  │   │
│  │  │ Security │  │ Performance │  │    LLM Insights    │  │   │
│  │  └──────────┘  └─────────────┘  └────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Vector Store (Memory/PostgreSQL)           │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                            │
                 ┌──────────┴──────────┐
                 │  Kubernetes Cluster │
                 └─────────────────────┘
```

## Installation

### Prerequisites

- Go 1.21+
- Temporal server (for workflow orchestration)
- PostgreSQL with pgvector extension (optional, for persistent storage)
- Ollama or OpenAI API (for LLM and embeddings)

### Build

```bash
# Clone the repository
git clone https://github.com/kubeanalysis/kubeanalysis
cd kubeanalysis

# Build
make build

# Run tests
make test
```

## Configuration

### Initialize Configuration

```bash
kubeanalysis init -c config.yaml
```

### Configuration File

```yaml
# Kubernetes connection
kubernetes:
  kubeconfig: ""  # Leave empty for in-cluster
  context: ""
  namespaces: []
  exclude_namespaces:
    - kube-system
    - kube-public
  qps: 50
  burst: 100

# Temporal workflow engine
temporal:
  host: "localhost:7233"
  namespace: "default"
  task_queue: "k8s-analysis-queue"
  worker_count: 4
  max_concurrent_activities: 10
  max_concurrent_workflows: 10

# LLM configuration
llm:
  provider: "ollama"  # ollama, openai, anthropic
  base_url: "http://localhost:11434"
  api_key: ""
  model: "qwen2.5:32b"
  max_tokens: 4096
  temperature: 0.1
  timeout: 120s

# Embedding configuration
embedder:
  provider: "ollama"
  base_url: "http://localhost:11434"
  api_key: ""
  model: "nomic-embed-text"
  dimension: 768
  batch_size: 32

# RAG configuration
rag:
  top_k: 10
  min_score: 0.5
  max_context_length: 8000
  chunk_size: 1000
  chunk_overlap: 200
  knowledge_path: "./knowledge"

# Analysis settings
analysis:
  interval: 15m
  types:
    - security
    - performance
    - prediction
  severity_threshold: "HIGH"
  enable_predictions: true

# Logging
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, console
  output: "stdout"
```

### Environment Variables

Configuration can be overridden using environment variables:

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | Path to kubeconfig file |
| `TEMPORAL_HOST` | Temporal server address |
| `TEMPORAL_NAMESPACE` | Temporal namespace |
| `LLM_PROVIDER` | LLM provider (ollama/openai/anthropic) |
| `LLM_BASE_URL` | LLM API base URL |
| `LLM_API_KEY` | LLM API key |
| `LLM_MODEL` | LLM model name |
| `EMBEDDER_PROVIDER` | Embedding provider |
| `EMBEDDER_BASE_URL` | Embedding API base URL |
| `EMBEDDER_API_KEY` | Embedding API key |

## Usage

### Start the Worker

```bash
# Start Temporal server (if not already running)
temporal server start-dev

# Start the analysis worker
kubeanalysis worker -c config.yaml
```

### Run Analysis

```bash
# Full analysis
kubeanalysis analyze -t full

# Security analysis on specific namespace
kubeanalysis analyze -t security -n production

# Performance analysis with JSON output
kubeanalysis analyze -t performance -o json
```

### Ingest Custom Knowledge

```bash
kubeanalysis ingest -c config.yaml --path ./my-knowledge
```

## Integration Details

### Vector Store

KubeAnalysis supports two vector store backends:

#### In-Memory (Default)

Suitable for development and testing:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/rag"

store := rag.NewInMemoryVectorStore()
```

#### PostgreSQL with pgvector

For production deployments with persistent storage:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/rag"

cfg := rag.VectorStoreConfig{
    Type:        rag.VectorStoreTypePostgres,
    PostgresURL: "postgres://user:pass@localhost:5432/kubeanalysis?sslmode=disable",
    TableName:   "vector_chunks",
    Dimension:   768,
    CreateTable: true,
    MaxConns:    10,
}

store, err := rag.NewVectorStoreFromConfig(cfg)
```

**PostgreSQL Setup:**

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- The table is created automatically when CreateTable is true
```

### Health Endpoints

KubeAnalysis provides Kubernetes-compatible health endpoints:

| Endpoint | Purpose | Description |
|----------|---------|-------------|
| `/live` | Liveness probe | Returns 200 if the application is running |
| `/ready` | Readiness probe | Returns 200 if ready to serve traffic |
| `/health` | Detailed health | Returns component-level health status |

**Usage:**

```go
import (
    "github.com/kubeanalysis/kubeanalysis/internal/observability"
    "github.com/kubeanalysis/kubeanalysis/internal/server"
)

// Create server with health endpoints
srv := server.NewServer(server.DefaultServerConfig(), "1.0.0")

// Register health checkers
srv.HealthManager().RegisterChecker("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})

srv.HealthManager().RegisterChecker("temporal", func(ctx context.Context) error {
    // Check Temporal connection
    return nil
})

// Mark as ready when initialization is complete
srv.HealthManager().SetReady(true)
```

**Health Response Example:**

```json
{
  "status": "healthy",
  "components": [
    {
      "name": "database",
      "status": "healthy",
      "last_check": "2024-01-15T10:30:00Z"
    },
    {
      "name": "temporal",
      "status": "healthy",
      "last_check": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

### Structured Logging

KubeAnalysis uses [zap](https://github.com/uber-go/zap) for high-performance structured logging:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/observability"

// Initialize logger
cfg := observability.LogConfig{
    Level:      "info",
    Format:     "json",    // or "console"
    OutputPath: "stdout",
}
observability.InitLogger(cfg)

// Get logger
logger := observability.Logger()

// Structured logging
logger.Info("Analysis started",
    zap.String("cluster", clusterID),
    zap.String("type", analysisType),
)

// Context-aware logging (includes trace/request IDs)
ctx = observability.WithRequestID(ctx, "req-123")
observability.LogFromContext(ctx).Info("Processing request")
```

### Prometheus Metrics

KubeAnalysis exports Prometheus metrics at `/metrics`:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/observability"

// Initialize metrics
metrics := observability.NewMetrics()

// Record analysis
metrics.RecordAnalysis("security", "success", 5.2)

// Record findings
metrics.RecordFinding("security", "CRITICAL")

// Record RAG queries
metrics.RecordRAGQuery("k8s-security", 10, 0.85, true)

// Record LLM requests
metrics.RecordLLMRequest("ollama", "qwen2.5:32b", true, 1.5)
```

**Available Metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `kubeanalysis_analyses_total` | Counter | Total analyses by type and status |
| `kubeanalysis_analysis_duration_seconds` | Histogram | Analysis duration |
| `kubeanalysis_findings_total` | Counter | Findings by type and severity |
| `kubeanalysis_rag_queries_total` | Counter | RAG queries by namespace |
| `kubeanalysis_rag_query_duration_seconds` | Histogram | RAG query duration |
| `kubeanalysis_llm_requests_total` | Counter | LLM requests by provider/model |
| `kubeanalysis_llm_request_duration_seconds` | Histogram | LLM request duration |
| `kubeanalysis_k8s_requests_total` | Counter | Kubernetes API requests |
| `kubeanalysis_rate_limit_requests_total` | Counter | Rate-limited requests |
| `kubeanalysis_component_health` | Gauge | Component health status |

### Rate Limiting

Token bucket rate limiting is available for API protection:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/observability"

// Configure rate limiter
cfg := observability.RateLimitConfig{
    Enabled:        true,
    RequestsPerSec: 10.0,
    BurstSize:      20,
    CleanupInterval: 5 * time.Minute,
}

rl := observability.NewRateLimiter(cfg)
defer rl.Close()

// Use as middleware
middleware := observability.RateLimitMiddleware(rl, observability.IPKeyFunc)
http.Handle("/api/", middleware(apiHandler))

// Or check manually
if !rl.AllowKey(clientIP) {
    http.Error(w, "Too many requests", http.StatusTooManyRequests)
    return
}

// Wait for token (with context timeout)
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
if err := rl.WaitKey(ctx, clientIP); err != nil {
    // Rate limited or context cancelled
}
```

### Graceful Shutdown

The server supports graceful shutdown with configurable timeout and hooks:

```go
import "github.com/kubeanalysis/kubeanalysis/internal/server"

// Create server
cfg := server.ServerConfig{
    Host:            "0.0.0.0",
    Port:            8080,
    ShutdownTimeout: 30 * time.Second,
}
srv := server.NewServer(cfg, "1.0.0")

// Register shutdown hooks
srv.OnShutdown(func(ctx context.Context) error {
    // Close database connections
    return db.Close()
})

srv.OnShutdown(func(ctx context.Context) error {
    // Stop Temporal worker
    worker.Stop()
    return nil
})

// Start with graceful shutdown (blocks until SIGINT/SIGTERM)
srv.StartWithGracefulShutdown()
```

**Shutdown sequence:**

1. Receive SIGINT or SIGTERM
2. Mark service as not ready (fails readiness probe)
3. Execute shutdown hooks in parallel
4. Drain existing connections
5. Exit cleanly

## Deployment

KubeAnalysis supports multiple deployment strategies for different environments.

### Docker

Build and run locally with Docker:

```bash
# Build the image
docker build -t kubeanalysis:latest .

# Run with Docker
docker run -d \
  --name kubeanalysis \
  -p 8080:8080 \
  -v ~/.kube/config:/home/kubeanalysis/.kube/config:ro \
  -v ./config.yaml:/config/config.yaml:ro \
  kubeanalysis:latest worker -c /config/config.yaml
```

### Docker Compose (Local Development)

The easiest way to get started with a complete development environment:

```bash
# Start all services (KubeAnalysis, Temporal, PostgreSQL, Ollama, Prometheus, Grafana)
docker-compose up -d

# View logs
docker-compose logs -f kubeanalysis

# Stop all services
docker-compose down
```

Services included:
- **KubeAnalysis** - Main application (port 8080)
- **Temporal** - Workflow orchestration (port 7233)
- **Temporal UI** - Workflow management UI (port 8088)
- **PostgreSQL** - Vector store with pgvector (port 5432)
- **Ollama** - Local LLM server (port 11434)
- **Prometheus** - Metrics collection (port 9090)
- **Grafana** - Dashboards (port 3000, admin/admin)

Pull LLM models after starting:

```bash
# Pull the LLM model
docker exec -it ollama ollama pull qwen2.5:32b

# Pull the embedding model
docker exec -it ollama ollama pull nomic-embed-text
```

### Helm Chart

For production Kubernetes deployments:

```bash
# Add the Helm repository (when published)
helm repo add kubeanalysis https://kubeanalysis.github.io/charts

# Install with default values
helm install kubeanalysis kubeanalysis/kubeanalysis

# Install with custom values
helm install kubeanalysis kubeanalysis/kubeanalysis \
  --namespace kubeanalysis \
  --create-namespace \
  --set replicaCount=3 \
  --set config.temporal.host=temporal:7233 \
  --set config.llm.provider=openai \
  --set secrets.llmApiKey=$OPENAI_API_KEY

# Install from local chart
helm install kubeanalysis ./deploy/helm/kubeanalysis -f values-production.yaml
```

Key Helm values:

| Value | Description | Default |
|-------|-------------|---------|
| `replicaCount` | Number of replicas | 2 |
| `config.temporal.host` | Temporal server address | temporal-frontend:7233 |
| `config.llm.provider` | LLM provider | ollama |
| `config.llm.model` | LLM model name | qwen2.5:32b |
| `vectorStore.type` | Vector store type | memory |
| `postgresql.enabled` | Enable PostgreSQL | false |
| `autoscaling.enabled` | Enable HPA | true |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | false |

### Kustomize

For GitOps-style deployments:

```bash
# Deploy to development
kubectl apply -k deploy/kustomize/overlays/dev

# Deploy to staging
kubectl apply -k deploy/kustomize/overlays/staging

# Deploy to production
kubectl apply -k deploy/kustomize/overlays/production

# Preview changes
kubectl diff -k deploy/kustomize/overlays/production
```

Environment differences:

| Environment | Replicas | CPU Limit | Memory Limit | Features |
|-------------|----------|-----------|--------------|----------|
| dev | 1 | 500m | 512Mi | Debug logging |
| staging | 2 | 1000m | 1Gi | Standard |
| production | 3 | 2000m | 2Gi | NetworkPolicy, ServiceMonitor |

### Manual Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubeanalysis
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: kubeanalysis
        image: ghcr.io/kubeanalysis/kubeanalysis:latest
        args: ["worker", "-c", "/config/config.yaml"]
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### ServiceMonitor for Prometheus

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kubeanalysis
spec:
  selector:
    matchLabels:
      app: kubeanalysis
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## CI/CD

The project includes GitHub Actions workflows for continuous integration and releases.

### CI Pipeline (`.github/workflows/ci.yaml`)

Runs on every push and PR:
- **Lint**: golangci-lint
- **Test**: Unit tests with coverage
- **Build**: Binary and Docker image
- **Helm Lint**: Validate Helm chart
- **Kustomize Validate**: Validate all overlays

### Release Pipeline (`.github/workflows/release.yaml`)

Triggered on version tags (e.g., `v1.0.0`):
- Builds multi-arch Docker images (amd64, arm64)
- Publishes to GitHub Container Registry
- Creates GitHub release with binaries
- Packages Helm chart

### Creating a Release

```bash
# Tag a release
git tag v1.0.0
git push origin v1.0.0

# This triggers the release workflow which:
# 1. Builds and pushes Docker image to ghcr.io
# 2. Creates GitHub release with binaries
# 3. Packages Helm chart
```

## Development

### Project Structure

```
kubeanalysis/
├── .github/
│   └── workflows/          # CI/CD pipelines
├── cmd/kubeanalysis/       # CLI entry point
├── deploy/
│   ├── grafana/            # Grafana dashboards and provisioning
│   ├── helm/kubeanalysis/  # Helm chart
│   ├── kustomize/          # Kustomize base and overlays
│   ├── postgres/           # PostgreSQL init scripts
│   └── prometheus/         # Prometheus configuration
├── internal/
│   ├── analyzer/           # Security and performance analyzers
│   ├── config/             # Configuration management
│   ├── k8s/                # Kubernetes client
│   ├── llm/                # LLM client and prompts
│   ├── observability/      # Logging, metrics, health, rate limiting
│   ├── rag/                # RAG engine, embedder, vector store
│   ├── server/             # HTTP server with graceful shutdown
│   └── temporal/           # Temporal workflows and activities
├── knowledge/              # Knowledge base documents
├── docker-compose.yaml     # Local development stack
├── Dockerfile              # Multi-stage container build
├── Makefile
└── README.md
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/observability/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Build
make build
```

## License

Apache License 2.0
