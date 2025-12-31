package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all application metrics.
type Metrics struct {
	// Analysis metrics
	AnalysesTotal      *prometheus.CounterVec
	AnalysisDuration   *prometheus.HistogramVec
	FindingsTotal      *prometheus.CounterVec
	AnalysisErrors     *prometheus.CounterVec

	// RAG metrics
	RAGQueriesTotal    *prometheus.CounterVec
	RAGQueryDuration   *prometheus.HistogramVec
	RAGDocumentsTotal  prometheus.Gauge
	EmbeddingDuration  *prometheus.HistogramVec

	// LLM metrics
	LLMRequestsTotal   *prometheus.CounterVec
	LLMRequestDuration *prometheus.HistogramVec
	LLMTokensUsed      *prometheus.CounterVec
	LLMErrors          *prometheus.CounterVec

	// Kubernetes client metrics
	K8sRequestsTotal   *prometheus.CounterVec
	K8sRequestDuration *prometheus.HistogramVec
	K8sErrors          *prometheus.CounterVec

	// Temporal workflow metrics
	WorkflowsStarted   *prometheus.CounterVec
	WorkflowsCompleted *prometheus.CounterVec
	WorkflowsFailed    *prometheus.CounterVec
	WorkflowDuration   *prometheus.HistogramVec

	// Rate limiting metrics
	RateLimitHits      *prometheus.CounterVec
	RateLimitRemaining *prometheus.GaugeVec

	// Health metrics
	HealthStatus       *prometheus.GaugeVec
}

var globalMetrics *Metrics

// InitMetrics initializes and registers all metrics.
func InitMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "kubeanalysis"
	}

	m := &Metrics{
		// Analysis metrics
		AnalysesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "analyses_total",
				Help:      "Total number of analyses performed",
			},
			[]string{"type", "status"},
		),
		AnalysisDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "analysis_duration_seconds",
				Help:      "Duration of analysis operations",
				Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
			},
			[]string{"type"},
		),
		FindingsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "findings_total",
				Help:      "Total number of findings",
			},
			[]string{"severity", "category"},
		),
		AnalysisErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "analysis_errors_total",
				Help:      "Total number of analysis errors",
			},
			[]string{"type", "error_type"},
		),

		// RAG metrics
		RAGQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rag_queries_total",
				Help:      "Total number of RAG queries",
			},
			[]string{"status"},
		),
		RAGQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "rag_query_duration_seconds",
				Help:      "Duration of RAG queries",
				Buckets:   prometheus.ExponentialBuckets(0.01, 2, 10),
			},
			[]string{},
		),
		RAGDocumentsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "rag_documents_total",
				Help:      "Total number of documents in the RAG store",
			},
		),
		EmbeddingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "embedding_duration_seconds",
				Help:      "Duration of embedding generation",
				Buckets:   prometheus.ExponentialBuckets(0.01, 2, 10),
			},
			[]string{"provider"},
		),

		// LLM metrics
		LLMRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "llm_requests_total",
				Help:      "Total number of LLM requests",
			},
			[]string{"provider", "model", "status"},
		),
		LLMRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "llm_request_duration_seconds",
				Help:      "Duration of LLM requests",
				Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
			},
			[]string{"provider", "model"},
		),
		LLMTokensUsed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "llm_tokens_used_total",
				Help:      "Total tokens used by LLM",
			},
			[]string{"provider", "model", "type"},
		),
		LLMErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "llm_errors_total",
				Help:      "Total LLM errors",
			},
			[]string{"provider", "error_type"},
		),

		// Kubernetes client metrics
		K8sRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "k8s_requests_total",
				Help:      "Total number of Kubernetes API requests",
			},
			[]string{"resource", "verb", "status"},
		),
		K8sRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "k8s_request_duration_seconds",
				Help:      "Duration of Kubernetes API requests",
				Buckets:   prometheus.ExponentialBuckets(0.01, 2, 10),
			},
			[]string{"resource", "verb"},
		),
		K8sErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "k8s_errors_total",
				Help:      "Total Kubernetes API errors",
			},
			[]string{"resource", "error_type"},
		),

		// Temporal workflow metrics
		WorkflowsStarted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_started_total",
				Help:      "Total workflows started",
			},
			[]string{"workflow_type"},
		),
		WorkflowsCompleted: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_completed_total",
				Help:      "Total workflows completed successfully",
			},
			[]string{"workflow_type"},
		),
		WorkflowsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_failed_total",
				Help:      "Total workflows failed",
			},
			[]string{"workflow_type", "error_type"},
		),
		WorkflowDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "workflow_duration_seconds",
				Help:      "Duration of workflow execution",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"workflow_type"},
		),

		// Rate limiting metrics
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_limit_hits_total",
				Help:      "Total rate limit hits",
			},
			[]string{"limiter"},
		),
		RateLimitRemaining: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "rate_limit_remaining",
				Help:      "Remaining rate limit tokens",
			},
			[]string{"limiter"},
		),

		// Health metrics
		HealthStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "health_status",
				Help:      "Health status of components (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),
	}

	globalMetrics = m
	return m
}

// M returns the global metrics instance.
func M() *Metrics {
	if globalMetrics == nil {
		return InitMetrics("")
	}
	return globalMetrics
}

// MetricsHandler returns the Prometheus HTTP handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordAnalysis records analysis metrics.
func (m *Metrics) RecordAnalysis(analysisType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		m.AnalysisErrors.WithLabelValues(analysisType, errorType(err)).Inc()
	}
	m.AnalysesTotal.WithLabelValues(analysisType, status).Inc()
	m.AnalysisDuration.WithLabelValues(analysisType).Observe(duration.Seconds())
}

// RecordFinding records a finding.
func (m *Metrics) RecordFinding(severity, category string) {
	m.FindingsTotal.WithLabelValues(severity, category).Inc()
}

// RecordRAGQuery records a RAG query.
func (m *Metrics) RecordRAGQuery(duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	m.RAGQueriesTotal.WithLabelValues(status).Inc()
	m.RAGQueryDuration.WithLabelValues().Observe(duration.Seconds())
}

// RecordLLMRequest records an LLM request.
func (m *Metrics) RecordLLMRequest(provider, model string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		m.LLMErrors.WithLabelValues(provider, errorType(err)).Inc()
	}
	m.LLMRequestsTotal.WithLabelValues(provider, model, status).Inc()
	m.LLMRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordK8sRequest records a Kubernetes API request.
func (m *Metrics) RecordK8sRequest(resource, verb string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		m.K8sErrors.WithLabelValues(resource, errorType(err)).Inc()
	}
	m.K8sRequestsTotal.WithLabelValues(resource, verb, status).Inc()
	m.K8sRequestDuration.WithLabelValues(resource, verb).Observe(duration.Seconds())
}

// RecordWorkflow records workflow metrics.
func (m *Metrics) RecordWorkflow(workflowType string, duration time.Duration, err error) {
	m.WorkflowsStarted.WithLabelValues(workflowType).Inc()
	if err != nil {
		m.WorkflowsFailed.WithLabelValues(workflowType, errorType(err)).Inc()
	} else {
		m.WorkflowsCompleted.WithLabelValues(workflowType).Inc()
	}
	m.WorkflowDuration.WithLabelValues(workflowType).Observe(duration.Seconds())
}

// SetHealthStatus sets the health status for a component.
func (m *Metrics) SetHealthStatus(component string, healthy bool) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	m.HealthStatus.WithLabelValues(component).Set(val)
}

func errorType(err error) string {
	if err == nil {
		return "none"
	}
	// Could add more specific error type detection here
	return "unknown"
}
