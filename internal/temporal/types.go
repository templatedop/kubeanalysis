// Package temporal provides Temporal workflow orchestration for Kubernetes analysis.
package temporal

import (
	"time"
)

// AnalysisType defines the type of analysis to perform.
type AnalysisType string

const (
	AnalysisTypeSecurity    AnalysisType = "security"
	AnalysisTypePerformance AnalysisType = "performance"
	AnalysisTypePrediction  AnalysisType = "prediction"
	AnalysisTypeFull        AnalysisType = "full"
)

// Severity represents the severity level of a finding.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// AnalysisInput represents input for an analysis workflow.
type AnalysisInput struct {
	ClusterID    string       `json:"cluster_id"`
	AnalysisType AnalysisType `json:"analysis_type"`
	Namespaces   []string     `json:"namespaces,omitempty"`
	Priority     string       `json:"priority,omitempty"`
	RequestedBy  string       `json:"requested_by,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
}

// AnalysisResult represents the result of an analysis workflow.
type AnalysisResult struct {
	ClusterID    string              `json:"cluster_id"`
	AnalysisType AnalysisType        `json:"analysis_type"`
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Duration     time.Duration       `json:"duration"`
	Status       string              `json:"status"`
	Findings     []Finding           `json:"findings,omitempty"`
	Security     *SecurityReport     `json:"security,omitempty"`
	Performance  *PerformanceReport  `json:"performance,omitempty"`
	Predictions  *PredictionReport   `json:"predictions,omitempty"`
	LLMInsights  *LLMInsights        `json:"llm_insights,omitempty"`
	Error        string              `json:"error,omitempty"`
}

// Finding represents a generic analysis finding.
type Finding struct {
	ID           string            `json:"id"`
	Severity     Severity          `json:"severity"`
	Category     string            `json:"category"`
	Resource     ResourceRef       `json:"resource"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Evidence     []string          `json:"evidence,omitempty"`
	Remediation  string            `json:"remediation,omitempty"`
	Framework    string            `json:"framework,omitempty"`
	ControlID    string            `json:"control_id,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ResourceRef references a Kubernetes resource.
type ResourceRef struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	APIVersion string `json:"api_version,omitempty"`
}

// SecurityReport contains security analysis results.
type SecurityReport struct {
	Timestamp       time.Time           `json:"timestamp"`
	Findings        []SecurityFinding   `json:"findings"`
	Summary         SecuritySummary     `json:"summary"`
	Compliance      ComplianceStatus    `json:"compliance"`
	Recommendations []string            `json:"recommendations,omitempty"`
}

// SecurityFinding represents a security-specific finding.
type SecurityFinding struct {
	Finding
	AttackVector    string   `json:"attack_vector,omitempty"`
	CVE             []string `json:"cve,omitempty"`
	CWE             string   `json:"cwe,omitempty"`
	Exploitability  string   `json:"exploitability,omitempty"`
}

// SecuritySummary summarizes security findings.
type SecuritySummary struct {
	TotalFindings    int            `json:"total_findings"`
	BySeverity       map[string]int `json:"by_severity"`
	ByCategory       map[string]int `json:"by_category"`
	CriticalCount    int            `json:"critical_count"`
	HighCount        int            `json:"high_count"`
	PostureScore     float64        `json:"posture_score"`
	RiskLevel        string         `json:"risk_level"`
}

// ComplianceStatus represents compliance status against frameworks.
type ComplianceStatus struct {
	NSACISA    FrameworkCompliance `json:"nsa_cisa,omitempty"`
	CIS        FrameworkCompliance `json:"cis,omitempty"`
	MITRE      FrameworkCompliance `json:"mitre,omitempty"`
	Custom     FrameworkCompliance `json:"custom,omitempty"`
}

// FrameworkCompliance represents compliance status for a framework.
type FrameworkCompliance struct {
	Framework       string   `json:"framework"`
	PassedControls  int      `json:"passed_controls"`
	FailedControls  int      `json:"failed_controls"`
	TotalControls   int      `json:"total_controls"`
	Score           float64  `json:"score"`
	Violations      []string `json:"violations,omitempty"`
}

// PerformanceReport contains performance analysis results.
type PerformanceReport struct {
	Timestamp       time.Time              `json:"timestamp"`
	Findings        []PerformanceFinding   `json:"findings"`
	Summary         PerformanceSummary     `json:"summary"`
	Recommendations []ResourceRecommendation `json:"recommendations,omitempty"`
}

// PerformanceFinding represents a performance-specific finding.
type PerformanceFinding struct {
	Finding
	Metrics         map[string]float64 `json:"metrics,omitempty"`
	Impact          string             `json:"impact,omitempty"`
	Threshold       float64            `json:"threshold,omitempty"`
	CurrentValue    float64            `json:"current_value,omitempty"`
}

// PerformanceSummary summarizes performance findings.
type PerformanceSummary struct {
	TotalFindings    int            `json:"total_findings"`
	BySeverity       map[string]int `json:"by_severity"`
	ByCategory       map[string]int `json:"by_category"`
	HealthScore      float64        `json:"health_score"`
	UtilizationScore float64        `json:"utilization_score"`
}

// ResourceRecommendation represents a resource sizing recommendation.
type ResourceRecommendation struct {
	Resource        ResourceRef `json:"resource"`
	CurrentCPU      string      `json:"current_cpu,omitempty"`
	CurrentMemory   string      `json:"current_memory,omitempty"`
	RecommendedCPU  string      `json:"recommended_cpu,omitempty"`
	RecommendedMemory string    `json:"recommended_memory,omitempty"`
	Reasoning       string      `json:"reasoning,omitempty"`
	SavingsEstimate string      `json:"savings_estimate,omitempty"`
}

// PredictionReport contains prediction analysis results.
type PredictionReport struct {
	Timestamp       time.Time      `json:"timestamp"`
	Predictions     []Prediction   `json:"predictions"`
	Summary         PredictionSummary `json:"summary"`
	ActionPlan      []ActionItem   `json:"action_plan,omitempty"`
}

// Prediction represents a capacity or failure prediction.
type Prediction struct {
	ID             string        `json:"id"`
	Type           string        `json:"type"` // capacity, failure, cost
	Severity       Severity      `json:"severity"`
	Resource       ResourceRef   `json:"resource"`
	Title          string        `json:"title"`
	Description    string        `json:"description,omitempty"`
	CurrentValue   float64       `json:"current_value,omitempty"`
	PredictedValue float64       `json:"predicted_value,omitempty"`
	Threshold      float64       `json:"threshold,omitempty"`
	TimeToEvent    time.Duration `json:"time_to_event,omitempty"`
	Confidence     float64       `json:"confidence"`
	Reasoning      string        `json:"reasoning,omitempty"`
	Recommendation string        `json:"recommendation,omitempty"`
}

// PredictionSummary summarizes predictions.
type PredictionSummary struct {
	TotalPredictions int            `json:"total_predictions"`
	BySeverity       map[string]int `json:"by_severity"`
	ByType           map[string]int `json:"by_type"`
	HighRiskCount    int            `json:"high_risk_count"`
	AverageConfidence float64       `json:"average_confidence"`
}

// ActionItem represents an action to take based on predictions.
type ActionItem struct {
	Prediction    string        `json:"prediction_id"`
	Action        string        `json:"action"`
	Deadline      time.Time     `json:"deadline,omitempty"`
	Effort        string        `json:"effort,omitempty"`
	Priority      int           `json:"priority"`
	Alternatives  []string      `json:"alternatives,omitempty"`
}

// LLMInsights contains LLM-generated insights.
type LLMInsights struct {
	ExecutiveSummary    string            `json:"executive_summary"`
	KeyRisks            []string          `json:"key_risks,omitempty"`
	PrioritizedActions  []string          `json:"prioritized_actions,omitempty"`
	CorrelatedFindings  []CorrelatedFinding `json:"correlated_findings,omitempty"`
	RootCauseAnalysis   string            `json:"root_cause_analysis,omitempty"`
	RemediationRoadmap  *RemediationRoadmap `json:"remediation_roadmap,omitempty"`
	ConfidenceLevel     float64           `json:"confidence_level"`
}

// CorrelatedFinding represents findings that are related.
type CorrelatedFinding struct {
	FindingIDs    []string `json:"finding_ids"`
	Relationship  string   `json:"relationship"`
	CombinedRisk  string   `json:"combined_risk,omitempty"`
	AttackChain   string   `json:"attack_chain,omitempty"`
}

// RemediationRoadmap provides a structured remediation plan.
type RemediationRoadmap struct {
	Immediate []RemediationAction `json:"immediate,omitempty"` // < 24 hours
	ShortTerm []RemediationAction `json:"short_term,omitempty"` // < 1 week
	LongTerm  []RemediationAction `json:"long_term,omitempty"` // ongoing
}

// RemediationAction represents a remediation action.
type RemediationAction struct {
	Action     string   `json:"action"`
	FindingIDs []string `json:"finding_ids,omitempty"`
	Effort     string   `json:"effort,omitempty"`
	Impact     string   `json:"impact,omitempty"`
}

// MonitoringConfig configures continuous monitoring.
type MonitoringConfig struct {
	ClusterID       string        `json:"cluster_id"`
	Interval        time.Duration `json:"interval"`
	AnalysisType    AnalysisType  `json:"analysis_type"`
	AlertThresholds AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds defines thresholds for alerting.
type AlertThresholds struct {
	CriticalCount int     `json:"critical_count"`
	HighCount     int     `json:"high_count"`
	ScoreThreshold float64 `json:"score_threshold"`
}

// EventAnalysisInput represents input for event-driven analysis.
type EventAnalysisInput struct {
	Event    K8sEvent `json:"event"`
	Priority string   `json:"priority,omitempty"`
}

// K8sEvent represents a Kubernetes event.
type K8sEvent struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Reason       string            `json:"reason"`
	Message      string            `json:"message"`
	Resource     ResourceRef       `json:"resource"`
	FirstSeen    time.Time         `json:"first_seen"`
	LastSeen     time.Time         `json:"last_seen"`
	Count        int               `json:"count"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// TriageResult represents the result of event triage.
type TriageResult struct {
	EventID         string   `json:"event_id"`
	Severity        Severity `json:"severity"`
	Category        string   `json:"category"`
	RequiresDeepAnalysis bool `json:"requires_deep_analysis"`
	Summary         string   `json:"summary"`
	SuggestedActions []string `json:"suggested_actions,omitempty"`
}
