package temporal

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/kubeanalysis/kubeanalysis/internal/rag"
)

// safeLogInfo logs info, recovering from panic if not in activity context.
func safeLogInfo(ctx context.Context, msg string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[INFO] "+msg+" %v", args...)
		}
	}()
	activity.GetLogger(ctx).Info(msg, args...)
}

// safeLogWarn logs warning, recovering from panic if not in activity context.
func safeLogWarn(ctx context.Context, msg string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WARN] "+msg+" %v", args...)
		}
	}()
	activity.GetLogger(ctx).Warn(msg, args...)
}

// safeRecordHeartbeat records heartbeat, recovering from panic if not in activity context.
func safeRecordHeartbeat(ctx context.Context, details interface{}) {
	defer func() { recover() }()
	activity.RecordHeartbeat(ctx, details)
}

// Activities holds dependencies for Temporal activities.
type Activities struct {
	ragEngine *rag.Engine
	// In production, would include:
	// k8sClient     *kubernetes.Clientset
	// prometheusClient *prometheus.Client
	// llmClient     *llm.Client
}

// NewActivities creates a new Activities instance.
func NewActivities(ragEngine *rag.Engine) *Activities {
	return &Activities{
		ragEngine: ragEngine,
	}
}

// CollectClusterStateActivity collects the current state of a Kubernetes cluster.
func (a *Activities) CollectClusterStateActivity(ctx context.Context, input AnalysisInput) (*ClusterState, error) {
	safeLogInfo(ctx, "Collecting cluster state", "clusterID", input.ClusterID)
	safeRecordHeartbeat(ctx, "collecting cluster state")

	// Simulate cluster state collection
	// In production, this would use k8s client-go to collect real state
	state := &ClusterState{
		ClusterID:   input.ClusterID,
		CollectedAt: time.Now(),
		Nodes:       []NodeInfo{},
		Pods:        []PodInfo{},
		Deployments: []DeploymentInfo{},
		Services:    []ServiceInfo{},
		Events:      []K8sEvent{},
		Namespaces:  input.Namespaces,
	}

	// Simulate some data for testing
	state.Nodes = append(state.Nodes, NodeInfo{
		Name:           "node-1",
		Status:         "Ready",
		CPUCapacity:    4000,
		MemoryCapacity: 16 * 1024 * 1024 * 1024,
		CPUUsage:       2500,
		MemoryUsage:    10 * 1024 * 1024 * 1024,
	})

	state.Pods = append(state.Pods, PodInfo{
		Name:         "test-pod",
		Namespace:    "default",
		Status:       "Running",
		RestartCount: 0,
		NodeName:     "node-1",
	})

	return state, nil
}

// SecurityAnalysisActivity performs security analysis on cluster state.
func (a *Activities) SecurityAnalysisActivity(ctx context.Context, state *ClusterState) (*SecurityReport, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Performing security analysis", "clusterID", state.ClusterID)

	safeRecordHeartbeat(ctx, "analyzing security")

	findings := []SecurityFinding{}

	// Analyze pods for security issues
	for _, pod := range state.Pods {
		// Check for privileged containers
		if pod.Privileged {
			findings = append(findings, SecurityFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("sec-%s-privileged", pod.Name),
					Severity:    SeverityCritical,
					Category:    "POD_SECURITY",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "Privileged container detected",
					Description: fmt.Sprintf("Pod %s is running with privileged mode", pod.Name),
					Evidence:    []string{"securityContext.privileged: true"},
					Remediation: "Remove privileged flag unless absolutely necessary",
					Framework:   "NSA",
					ControlID:   "POD-SEC-001",
					Timestamp:   time.Now(),
				},
				AttackVector:   "Container escape",
				Exploitability: "High",
			})
		}

		// Check for root user
		if pod.RunAsRoot {
			findings = append(findings, SecurityFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("sec-%s-root", pod.Name),
					Severity:    SeverityHigh,
					Category:    "POD_SECURITY",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "Container running as root",
					Description: fmt.Sprintf("Pod %s is running as root user", pod.Name),
					Remediation: "Set runAsNonRoot: true and specify a non-root runAsUser",
					Framework:   "CIS",
					ControlID:   "5.2.6",
					Timestamp:   time.Now(),
				},
				Exploitability: "Medium",
			})
		}

		// Check for missing network policies
		if !pod.HasNetworkPolicy {
			findings = append(findings, SecurityFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("sec-%s-netpol", pod.Name),
					Severity:    SeverityMedium,
					Category:    "NETWORK",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "No network policy applied",
					Description: fmt.Sprintf("Pod %s has no network policy restricting traffic", pod.Name),
					Remediation: "Apply a NetworkPolicy to restrict ingress and egress traffic",
					Framework:   "NSA",
					ControlID:   "NET-001",
					Timestamp:   time.Now(),
				},
			})
		}
	}

	// Generate summary
	summary := SecuritySummary{
		TotalFindings: len(findings),
		BySeverity:    make(map[string]int),
		ByCategory:    make(map[string]int),
	}

	for _, f := range findings {
		summary.BySeverity[string(f.Severity)]++
		summary.ByCategory[f.Category]++
		if f.Severity == SeverityCritical {
			summary.CriticalCount++
		} else if f.Severity == SeverityHigh {
			summary.HighCount++
		}
	}

	// Calculate posture score (100 - weighted findings)
	score := 100.0
	score -= float64(summary.CriticalCount) * 20
	score -= float64(summary.HighCount) * 10
	score -= float64(summary.BySeverity["MEDIUM"]) * 5
	score -= float64(summary.BySeverity["LOW"]) * 2
	if score < 0 {
		score = 0
	}
	summary.PostureScore = score

	if score >= 80 {
		summary.RiskLevel = "LOW"
	} else if score >= 60 {
		summary.RiskLevel = "MEDIUM"
	} else if score >= 40 {
		summary.RiskLevel = "HIGH"
	} else {
		summary.RiskLevel = "CRITICAL"
	}

	return &SecurityReport{
		Timestamp: time.Now(),
		Findings:  findings,
		Summary:   summary,
	}, nil
}

// PerformanceAnalysisActivity performs performance analysis on cluster state.
func (a *Activities) PerformanceAnalysisActivity(ctx context.Context, state *ClusterState) (*PerformanceReport, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Performing performance analysis", "clusterID", state.ClusterID)

	safeRecordHeartbeat(ctx, "analyzing performance")

	findings := []PerformanceFinding{}

	// Analyze nodes for resource issues
	for _, node := range state.Nodes {
		cpuUtilization := float64(node.CPUUsage) / float64(node.CPUCapacity) * 100
		memUtilization := float64(node.MemoryUsage) / float64(node.MemoryCapacity) * 100

		if cpuUtilization > 85 {
			findings = append(findings, PerformanceFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("perf-%s-cpu", node.Name),
					Severity:    SeverityHigh,
					Category:    "RESOURCE",
					Resource:    ResourceRef{Kind: "Node", Name: node.Name},
					Title:       "High CPU utilization on node",
					Description: fmt.Sprintf("Node %s CPU utilization is %.1f%%", node.Name, cpuUtilization),
					Remediation: "Scale cluster or optimize workloads",
					Timestamp:   time.Now(),
				},
				Metrics: map[string]float64{
					"cpu_utilization_pct": cpuUtilization,
					"cpu_usage":           float64(node.CPUUsage),
					"cpu_capacity":        float64(node.CPUCapacity),
				},
				Impact:       "Application latency and scheduling issues",
				Threshold:    85,
				CurrentValue: cpuUtilization,
			})
		}

		if memUtilization > 80 {
			findings = append(findings, PerformanceFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("perf-%s-mem", node.Name),
					Severity:    SeverityHigh,
					Category:    "RESOURCE",
					Resource:    ResourceRef{Kind: "Node", Name: node.Name},
					Title:       "High memory utilization on node",
					Description: fmt.Sprintf("Node %s memory utilization is %.1f%%", node.Name, memUtilization),
					Remediation: "Add nodes or reduce memory usage",
					Timestamp:   time.Now(),
				},
				Metrics: map[string]float64{
					"memory_utilization_pct": memUtilization,
					"memory_usage":           float64(node.MemoryUsage),
					"memory_capacity":        float64(node.MemoryCapacity),
				},
				Impact:       "Risk of OOM kills and pod evictions",
				Threshold:    80,
				CurrentValue: memUtilization,
			})
		}
	}

	// Analyze pods for health issues
	for _, pod := range state.Pods {
		if pod.RestartCount > 5 {
			findings = append(findings, PerformanceFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("perf-%s-restarts", pod.Name),
					Severity:    SeverityHigh,
					Category:    "HEALTH",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "Pod restart loop detected",
					Description: fmt.Sprintf("Pod %s has restarted %d times", pod.Name, pod.RestartCount),
					Remediation: "Check logs and events for root cause",
					Timestamp:   time.Now(),
				},
				Metrics: map[string]float64{
					"restart_count": float64(pod.RestartCount),
				},
				Impact: "Service instability",
			})
		}

		if pod.Status == "CrashLoopBackOff" {
			findings = append(findings, PerformanceFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("perf-%s-crash", pod.Name),
					Severity:    SeverityCritical,
					Category:    "HEALTH",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "CrashLoopBackOff detected",
					Description: fmt.Sprintf("Pod %s is in CrashLoopBackOff", pod.Name),
					Remediation: "Check container logs and fix application error",
					Timestamp:   time.Now(),
				},
				Impact: "Service unavailable",
			})
		}

		if pod.Status == "Pending" {
			findings = append(findings, PerformanceFinding{
				Finding: Finding{
					ID:          fmt.Sprintf("perf-%s-pending", pod.Name),
					Severity:    SeverityHigh,
					Category:    "HEALTH",
					Resource:    ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:       "Pod stuck in Pending state",
					Description: fmt.Sprintf("Pod %s cannot be scheduled", pod.Name),
					Remediation: "Check node resources, taints, and affinity rules",
					Timestamp:   time.Now(),
				},
				Impact: "Workload not running",
			})
		}
	}

	// Generate summary
	summary := PerformanceSummary{
		TotalFindings: len(findings),
		BySeverity:    make(map[string]int),
		ByCategory:    make(map[string]int),
	}

	for _, f := range findings {
		summary.BySeverity[string(f.Severity)]++
		summary.ByCategory[f.Category]++
	}

	// Calculate health score
	score := 100.0
	score -= float64(summary.BySeverity["CRITICAL"]) * 25
	score -= float64(summary.BySeverity["HIGH"]) * 15
	score -= float64(summary.BySeverity["MEDIUM"]) * 5
	if score < 0 {
		score = 0
	}
	summary.HealthScore = score

	return &PerformanceReport{
		Timestamp: time.Now(),
		Findings:  findings,
		Summary:   summary,
	}, nil
}

// PredictionAnalysisActivity performs predictive analysis.
func (a *Activities) PredictionAnalysisActivity(ctx context.Context, state *ClusterState) (*PredictionReport, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Performing prediction analysis", "clusterID", state.ClusterID)

	safeRecordHeartbeat(ctx, "generating predictions")

	predictions := []Prediction{}

	// Analyze nodes for capacity predictions
	for _, node := range state.Nodes {
		cpuUtilization := float64(node.CPUUsage) / float64(node.CPUCapacity) * 100
		memUtilization := float64(node.MemoryUsage) / float64(node.MemoryCapacity) * 100

		// Predict CPU exhaustion
		if cpuUtilization > 70 {
			predictions = append(predictions, Prediction{
				ID:             fmt.Sprintf("pred-%s-cpu", node.Name),
				Type:           "capacity",
				Severity:       SeverityHigh,
				Resource:       ResourceRef{Kind: "Node", Name: node.Name},
				Title:          "Node CPU exhaustion predicted",
				Description:    fmt.Sprintf("Based on current usage (%.1f%%), node may run out of CPU capacity", cpuUtilization),
				CurrentValue:   cpuUtilization,
				PredictedValue: 95,
				Threshold:      90,
				TimeToEvent:    24 * time.Hour, // Simplified prediction
				Confidence:     0.75,
				Recommendation: "Scale cluster or optimize CPU-intensive workloads",
			})
		}

		// Predict memory exhaustion
		if memUtilization > 70 {
			predictions = append(predictions, Prediction{
				ID:             fmt.Sprintf("pred-%s-mem", node.Name),
				Type:           "capacity",
				Severity:       SeverityCritical,
				Resource:       ResourceRef{Kind: "Node", Name: node.Name},
				Title:          "Node memory exhaustion predicted",
				Description:    fmt.Sprintf("Based on current usage (%.1f%%), node may run out of memory", memUtilization),
				CurrentValue:   memUtilization,
				PredictedValue: 95,
				Threshold:      90,
				TimeToEvent:    12 * time.Hour,
				Confidence:     0.80,
				Recommendation: "Add nodes or reduce memory usage",
			})
		}
	}

	// Predict pod failures based on restart patterns
	for _, pod := range state.Pods {
		if pod.RestartCount > 2 {
			predictions = append(predictions, Prediction{
				ID:             fmt.Sprintf("pred-%s-failure", pod.Name),
				Type:           "failure",
				Severity:       SeverityHigh,
				Resource:       ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
				Title:          "Pod failure likelihood increasing",
				Description:    fmt.Sprintf("Pod has restarted %d times, indicating instability", pod.RestartCount),
				CurrentValue:   float64(pod.RestartCount),
				Confidence:     0.65,
				Reasoning:      "Restart rate shows upward trend",
				Recommendation: "Investigate root cause before complete failure",
			})
		}
	}

	// Generate summary
	summary := PredictionSummary{
		TotalPredictions: len(predictions),
		BySeverity:       make(map[string]int),
		ByType:           make(map[string]int),
	}

	var totalConfidence float64
	for _, p := range predictions {
		summary.BySeverity[string(p.Severity)]++
		summary.ByType[p.Type]++
		if p.Severity == SeverityCritical || p.Severity == SeverityHigh {
			summary.HighRiskCount++
		}
		totalConfidence += p.Confidence
	}
	if len(predictions) > 0 {
		summary.AverageConfidence = totalConfidence / float64(len(predictions))
	}

	return &PredictionReport{
		Timestamp:   time.Now(),
		Predictions: predictions,
		Summary:     summary,
	}, nil
}

// FetchRAGContextActivity retrieves relevant context from the RAG knowledge base.
func (a *Activities) FetchRAGContextActivity(ctx context.Context, input RAGQueryInput) (*rag.RAGContext, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Fetching RAG context", "query", input.Query)

	safeRecordHeartbeat(ctx, "querying knowledge base")

	if a.ragEngine == nil {
		return &rag.RAGContext{
			Query:        input.Query,
			Results:      []rag.SearchResult{},
			TotalResults: 0,
		}, nil
	}

	return a.ragEngine.Query(ctx, input.Query, input.TopK)
}

// LLMEnhanceAnalysisActivity uses LLM to enhance analysis results.
func (a *Activities) LLMEnhanceAnalysisActivity(ctx context.Context, input LLMAnalysisInput) (*LLMInsights, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Enhancing analysis with LLM")

	safeRecordHeartbeat(ctx, "generating LLM insights")

	// In production, this would call the LLM API
	// For now, generate structured insights based on findings

	insights := &LLMInsights{
		ConfidenceLevel: 0.85,
	}

	// Generate executive summary
	var criticalCount, highCount int
	if input.Security != nil {
		criticalCount += input.Security.Summary.CriticalCount
		highCount += input.Security.Summary.HighCount
	}
	if input.Performance != nil {
		criticalCount += input.Performance.Summary.BySeverity["CRITICAL"]
		highCount += input.Performance.Summary.BySeverity["HIGH"]
	}

	if criticalCount > 0 {
		insights.ExecutiveSummary = fmt.Sprintf(
			"CRITICAL: Cluster requires immediate attention. Found %d critical and %d high severity issues. "+
				"Security posture is at risk and performance may be degraded.",
			criticalCount, highCount,
		)
		insights.KeyRisks = append(insights.KeyRisks, "Multiple critical security vulnerabilities detected")
	} else if highCount > 0 {
		insights.ExecutiveSummary = fmt.Sprintf(
			"HIGH PRIORITY: Cluster has %d high severity issues that should be addressed promptly. "+
				"No critical issues found but proactive remediation recommended.",
			highCount,
		)
	} else {
		insights.ExecutiveSummary = "Cluster is in good health. Minor issues detected that can be addressed during regular maintenance."
	}

	// Generate prioritized actions
	if input.Security != nil && input.Security.Summary.CriticalCount > 0 {
		insights.PrioritizedActions = append(insights.PrioritizedActions,
			"1. Address privileged container issues immediately",
			"2. Apply network policies to isolate workloads",
			"3. Implement pod security standards",
		)
	}

	// Generate remediation roadmap
	insights.RemediationRoadmap = &RemediationRoadmap{
		Immediate: []RemediationAction{
			{Action: "Disable privileged containers", Effort: "LOW"},
			{Action: "Apply default deny network policies", Effort: "MEDIUM"},
		},
		ShortTerm: []RemediationAction{
			{Action: "Implement pod security admission", Effort: "MEDIUM"},
			{Action: "Review and reduce RBAC permissions", Effort: "HIGH"},
		},
		LongTerm: []RemediationAction{
			{Action: "Adopt GitOps for security configurations", Effort: "HIGH"},
		},
	}

	return insights, nil
}

// SendAlertsActivity sends alerts for critical findings.
func (a *Activities) SendAlertsActivity(ctx context.Context, result *AnalysisResult) error {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Sending alerts", "clusterID", result.ClusterID)

	safeRecordHeartbeat(ctx, "sending alerts")

	// In production, this would integrate with:
	// - Slack
	// - PagerDuty
	// - Email
	// - Custom webhooks

	// For now, just log the alert
	criticalCount := 0
	if result.Security != nil {
		criticalCount += result.Security.Summary.CriticalCount
	}
	if result.Performance != nil {
		criticalCount += result.Performance.Summary.BySeverity["CRITICAL"]
	}

	if criticalCount > 0 {
		safeLogWarn(ctx, "ALERT: Critical findings detected",
			"clusterID", result.ClusterID,
			"criticalCount", criticalCount,
		)
	}

	return nil
}

// StoreAnalysisResultActivity stores analysis results.
func (a *Activities) StoreAnalysisResultActivity(ctx context.Context, result *AnalysisResult) error {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Storing analysis result", "clusterID", result.ClusterID)

	safeRecordHeartbeat(ctx, "storing results")

	// In production, this would store to:
	// - Database
	// - Object storage
	// - Time series database

	return nil
}

// TriageEventActivity performs quick triage on a Kubernetes event.
func (a *Activities) TriageEventActivity(ctx context.Context, event K8sEvent) (*TriageResult, error) {
	safeLogInfo(ctx, "Starting activity")
	safeLogInfo(ctx, "Triaging event", "eventID", event.ID, "reason", event.Reason)

	safeRecordHeartbeat(ctx, "triaging event")

	result := &TriageResult{
		EventID: event.ID,
	}

	// Determine severity based on event type and reason
	switch event.Reason {
	case "OOMKilled", "OOMKilling":
		result.Severity = SeverityCritical
		result.Category = "RESOURCE"
		result.RequiresDeepAnalysis = true
		result.Summary = "Container killed due to out-of-memory condition"
		result.SuggestedActions = []string{
			"Increase memory limits",
			"Investigate memory leak",
			"Add memory-based HPA",
		}
	case "CrashLoopBackOff":
		result.Severity = SeverityCritical
		result.Category = "HEALTH"
		result.RequiresDeepAnalysis = true
		result.Summary = "Container is in crash loop"
		result.SuggestedActions = []string{
			"Check container logs",
			"Verify application configuration",
			"Check resource limits",
		}
	case "FailedScheduling":
		result.Severity = SeverityHigh
		result.Category = "SCHEDULING"
		result.RequiresDeepAnalysis = true
		result.Summary = "Pod cannot be scheduled"
		result.SuggestedActions = []string{
			"Check cluster capacity",
			"Review node selectors and taints",
			"Check resource requests",
		}
	case "Unhealthy":
		result.Severity = SeverityHigh
		result.Category = "HEALTH"
		result.RequiresDeepAnalysis = false
		result.Summary = "Pod health check failing"
		result.SuggestedActions = []string{
			"Check probe configuration",
			"Verify application health endpoint",
		}
	case "BackOff":
		result.Severity = SeverityMedium
		result.Category = "HEALTH"
		result.RequiresDeepAnalysis = false
		result.Summary = "Container startup backoff"
		result.SuggestedActions = []string{
			"Check image pull status",
			"Verify container command",
		}
	default:
		result.Severity = SeverityInfo
		result.Category = "GENERAL"
		result.RequiresDeepAnalysis = false
		result.Summary = event.Message
	}

	return result, nil
}

// RAGQueryInput represents input for RAG queries.
type RAGQueryInput struct {
	Query   string            `json:"query"`
	TopK    int               `json:"top_k"`
	Filters map[string]string `json:"filters,omitempty"`
}

// LLMAnalysisInput represents input for LLM analysis.
type LLMAnalysisInput struct {
	Security    *SecurityReport    `json:"security,omitempty"`
	Performance *PerformanceReport `json:"performance,omitempty"`
	Predictions *PredictionReport  `json:"predictions,omitempty"`
	RAGContext  *rag.RAGContext    `json:"rag_context,omitempty"`
}

// ClusterState represents the state of a Kubernetes cluster.
type ClusterState struct {
	ClusterID   string           `json:"cluster_id"`
	CollectedAt time.Time        `json:"collected_at"`
	Nodes       []NodeInfo       `json:"nodes"`
	Pods        []PodInfo        `json:"pods"`
	Deployments []DeploymentInfo `json:"deployments"`
	Services    []ServiceInfo    `json:"services"`
	Events      []K8sEvent       `json:"events"`
	Namespaces  []string         `json:"namespaces"`
}

// NodeInfo represents information about a Kubernetes node.
type NodeInfo struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	CPUCapacity    int64  `json:"cpu_capacity"`
	MemoryCapacity int64  `json:"memory_capacity"`
	CPUUsage       int64  `json:"cpu_usage"`
	MemoryUsage    int64  `json:"memory_usage"`
}

// PodInfo represents information about a Kubernetes pod.
type PodInfo struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	Status           string `json:"status"`
	RestartCount     int    `json:"restart_count"`
	NodeName         string `json:"node_name"`
	Privileged       bool   `json:"privileged"`
	RunAsRoot        bool   `json:"run_as_root"`
	HasNetworkPolicy bool   `json:"has_network_policy"`
}

// DeploymentInfo represents information about a Kubernetes deployment.
type DeploymentInfo struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Replicas          int32  `json:"replicas"`
	ReadyReplicas     int32  `json:"ready_replicas"`
	AvailableReplicas int32  `json:"available_replicas"`
}

// ServiceInfo represents information about a Kubernetes service.
type ServiceInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	ClusterIP string `json:"cluster_ip"`
}
