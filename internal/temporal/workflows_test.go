package temporal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestK8sAnalysisWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register activity implementations
	env.RegisterActivity((&Activities{}).CollectClusterStateActivity)
	env.RegisterActivity((&Activities{}).SecurityAnalysisActivity)
	env.RegisterActivity((&Activities{}).PerformanceAnalysisActivity)
	env.RegisterActivity((&Activities{}).PredictionAnalysisActivity)
	env.RegisterActivity((&Activities{}).FetchRAGContextActivity)
	env.RegisterActivity((&Activities{}).LLMEnhanceAnalysisActivity)
	env.RegisterActivity((&Activities{}).SendAlertsActivity)
	env.RegisterActivity((&Activities{}).StoreAnalysisResultActivity)

	// Mock activities
	env.OnActivity((&Activities{}).CollectClusterStateActivity, mock.Anything, mock.Anything).Return(&ClusterState{
		ClusterID:   "test-cluster",
		CollectedAt: time.Now(),
		Nodes: []NodeInfo{
			{Name: "node-1", Status: "Ready", CPUCapacity: 4000, MemoryCapacity: 16 * 1024 * 1024 * 1024},
		},
		Pods: []PodInfo{
			{Name: "test-pod", Namespace: "default", Status: "Running"},
		},
	}, nil)

	env.OnActivity((&Activities{}).SecurityAnalysisActivity, mock.Anything, mock.Anything).Return(&SecurityReport{
		Timestamp: time.Now(),
		Findings:  []SecurityFinding{},
		Summary: SecuritySummary{
			TotalFindings: 0,
			PostureScore:  100,
			RiskLevel:     "LOW",
		},
	}, nil)

	env.OnActivity((&Activities{}).PerformanceAnalysisActivity, mock.Anything, mock.Anything).Return(&PerformanceReport{
		Timestamp: time.Now(),
		Findings:  []PerformanceFinding{},
		Summary: PerformanceSummary{
			TotalFindings: 0,
			HealthScore:   100,
		},
	}, nil)

	env.OnActivity((&Activities{}).PredictionAnalysisActivity, mock.Anything, mock.Anything).Return(&PredictionReport{
		Timestamp:   time.Now(),
		Predictions: []Prediction{},
		Summary:     PredictionSummary{TotalPredictions: 0},
	}, nil)

	env.OnActivity((&Activities{}).FetchRAGContextActivity, mock.Anything, mock.Anything).Return(nil, nil)

	env.OnActivity((&Activities{}).LLMEnhanceAnalysisActivity, mock.Anything, mock.Anything).Return(&LLMInsights{
		ExecutiveSummary: "Cluster is healthy",
		ConfidenceLevel:  0.9,
	}, nil)

	env.OnActivity((&Activities{}).SendAlertsActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity((&Activities{}).StoreAnalysisResultActivity, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := AnalysisInput{
		ClusterID:    "test-cluster",
		AnalysisType: AnalysisTypeFull,
	}

	env.ExecuteWorkflow(K8sAnalysisWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result AnalysisResult
	require.NoError(t, env.GetWorkflowResult(&result))

	require.Equal(t, "test-cluster", result.ClusterID)
	require.Equal(t, AnalysisTypeFull, result.AnalysisType)
	require.Equal(t, "completed", result.Status)
	require.NotNil(t, result.Security)
	require.NotNil(t, result.Performance)
	require.NotNil(t, result.Predictions)
}

func TestSecurityScanWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register activities
	env.RegisterActivity((&Activities{}).CollectClusterStateActivity)
	env.RegisterActivity((&Activities{}).SecurityAnalysisActivity)
	env.RegisterActivity((&Activities{}).FetchRAGContextActivity)
	env.RegisterActivity((&Activities{}).LLMEnhanceAnalysisActivity)

	// Mock activities
	env.OnActivity((&Activities{}).CollectClusterStateActivity, mock.Anything, mock.Anything).Return(&ClusterState{
		ClusterID: "test-cluster",
		Pods: []PodInfo{
			{
				Name:       "vulnerable-pod",
				Namespace:  "default",
				Privileged: true,
				RunAsRoot:  true,
			},
		},
	}, nil)

	env.OnActivity((&Activities{}).SecurityAnalysisActivity, mock.Anything, mock.Anything).Return(&SecurityReport{
		Timestamp: time.Now(),
		Findings: []SecurityFinding{
			{
				Finding: Finding{
					ID:       "sec-1",
					Severity: SeverityCritical,
					Title:    "Privileged container",
				},
			},
		},
		Summary: SecuritySummary{
			TotalFindings: 1,
			CriticalCount: 1,
			PostureScore:  60,
			RiskLevel:     "HIGH",
		},
	}, nil)

	env.OnActivity((&Activities{}).FetchRAGContextActivity, mock.Anything, mock.Anything).Return(nil, nil)
	env.OnActivity((&Activities{}).LLMEnhanceAnalysisActivity, mock.Anything, mock.Anything).Return(&LLMInsights{
		ExecutiveSummary: "Critical security issues found",
		PrioritizedActions: []string{
			"Remove privileged containers",
		},
	}, nil)

	// Execute workflow
	input := AnalysisInput{
		ClusterID:    "test-cluster",
		AnalysisType: AnalysisTypeSecurity,
	}

	env.ExecuteWorkflow(SecurityScanWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result SecurityReport
	require.NoError(t, env.GetWorkflowResult(&result))

	require.Equal(t, 1, len(result.Findings))
	require.Equal(t, SeverityCritical, result.Findings[0].Severity)
}

func TestEventAnalysisWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Register activities
	env.RegisterActivity((&Activities{}).TriageEventActivity)
	env.RegisterActivity((&Activities{}).CollectClusterStateActivity)
	env.RegisterActivity((&Activities{}).SecurityAnalysisActivity)
	env.RegisterActivity((&Activities{}).PerformanceAnalysisActivity)
	env.RegisterActivity((&Activities{}).SendAlertsActivity)

	// Mock triage activity
	env.OnActivity((&Activities{}).TriageEventActivity, mock.Anything, mock.Anything).Return(&TriageResult{
		EventID:             "event-1",
		Severity:            SeverityCritical,
		Category:            "HEALTH",
		RequiresDeepAnalysis: true,
		Summary:             "Container crashed",
	}, nil)

	env.OnActivity((&Activities{}).CollectClusterStateActivity, mock.Anything, mock.Anything).Return(&ClusterState{
		ClusterID: "test-cluster",
	}, nil)

	env.OnActivity((&Activities{}).SecurityAnalysisActivity, mock.Anything, mock.Anything).Return(&SecurityReport{
		Summary: SecuritySummary{CriticalCount: 0},
	}, nil)

	env.OnActivity((&Activities{}).PerformanceAnalysisActivity, mock.Anything, mock.Anything).Return(&PerformanceReport{
		Summary: PerformanceSummary{},
	}, nil)

	env.OnActivity((&Activities{}).SendAlertsActivity, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := EventAnalysisInput{
		Event: K8sEvent{
			ID:      "event-1",
			Type:    "Warning",
			Reason:  "CrashLoopBackOff",
			Message: "Container crashed",
			Resource: ResourceRef{
				Kind:      "Pod",
				Name:      "test-pod",
				Namespace: "default",
			},
		},
	}

	env.ExecuteWorkflow(EventAnalysisWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result AnalysisResult
	require.NoError(t, env.GetWorkflowResult(&result))

	require.Equal(t, "completed", result.Status)
	require.Len(t, result.Findings, 1)
	require.Equal(t, SeverityCritical, result.Findings[0].Severity)
}

func TestBuildRAGQuery(t *testing.T) {
	tests := []struct {
		name        string
		security    *SecurityReport
		performance *PerformanceReport
		predictions *PredictionReport
		expectEmpty bool
	}{
		{
			name:        "all nil",
			expectEmpty: false, // Should return default query
		},
		{
			name: "critical security",
			security: &SecurityReport{
				Summary: SecuritySummary{CriticalCount: 1},
			},
			expectEmpty: false,
		},
		{
			name: "critical performance",
			performance: &PerformanceReport{
				Summary: PerformanceSummary{
					BySeverity: map[string]int{"CRITICAL": 1},
				},
			},
			expectEmpty: false,
		},
		{
			name: "high risk predictions",
			predictions: &PredictionReport{
				Summary: PredictionSummary{HighRiskCount: 1},
			},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildRAGQuery(tt.security, tt.performance, tt.predictions)
			if tt.expectEmpty && query != "" {
				t.Errorf("expected empty query, got %s", query)
			}
			if !tt.expectEmpty && query == "" {
				t.Error("expected non-empty query")
			}
		})
	}
}

func TestHasCriticalFindings(t *testing.T) {
	tests := []struct {
		name     string
		result   *AnalysisResult
		expected bool
	}{
		{
			name:     "empty result",
			result:   &AnalysisResult{},
			expected: false,
		},
		{
			name: "critical security",
			result: &AnalysisResult{
				Security: &SecurityReport{
					Summary: SecuritySummary{CriticalCount: 1},
				},
			},
			expected: true,
		},
		{
			name: "critical performance",
			result: &AnalysisResult{
				Performance: &PerformanceReport{
					Summary: PerformanceSummary{
						BySeverity: map[string]int{"CRITICAL": 1},
					},
				},
			},
			expected: true,
		},
		{
			name: "critical predictions",
			result: &AnalysisResult{
				Predictions: &PredictionReport{
					Summary: PredictionSummary{
						BySeverity: map[string]int{"CRITICAL": 1},
					},
				},
			},
			expected: true,
		},
		{
			name: "no critical",
			result: &AnalysisResult{
				Security: &SecurityReport{
					Summary: SecuritySummary{CriticalCount: 0, HighCount: 5},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCriticalFindings(tt.result)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAggregateFindings(t *testing.T) {
	security := &SecurityReport{
		Findings: []SecurityFinding{
			{Finding: Finding{ID: "sec-1", Severity: SeverityCritical}},
			{Finding: Finding{ID: "sec-2", Severity: SeverityHigh}},
		},
	}

	performance := &PerformanceReport{
		Findings: []PerformanceFinding{
			{Finding: Finding{ID: "perf-1", Severity: SeverityMedium}},
		},
	}

	predictions := &PredictionReport{
		Predictions: []Prediction{
			{ID: "pred-1", Severity: SeverityLow, Title: "Prediction"},
		},
	}

	findings := aggregateFindings(security, performance, predictions)

	if len(findings) != 4 {
		t.Errorf("expected 4 findings, got %d", len(findings))
	}

	// Verify finding IDs
	ids := make(map[string]bool)
	for _, f := range findings {
		ids[f.ID] = true
	}

	for _, id := range []string{"sec-1", "sec-2", "perf-1", "pred-1"} {
		if !ids[id] {
			t.Errorf("missing finding ID: %s", id)
		}
	}
}

func TestAnalysisTypes(t *testing.T) {
	// Verify analysis type constants
	if AnalysisTypeSecurity != "security" {
		t.Error("AnalysisTypeSecurity should be 'security'")
	}
	if AnalysisTypePerformance != "performance" {
		t.Error("AnalysisTypePerformance should be 'performance'")
	}
	if AnalysisTypePrediction != "prediction" {
		t.Error("AnalysisTypePrediction should be 'prediction'")
	}
	if AnalysisTypeFull != "full" {
		t.Error("AnalysisTypeFull should be 'full'")
	}
}

func TestSeverityTypes(t *testing.T) {
	// Verify severity constants
	if SeverityCritical != "CRITICAL" {
		t.Error("SeverityCritical should be 'CRITICAL'")
	}
	if SeverityHigh != "HIGH" {
		t.Error("SeverityHigh should be 'HIGH'")
	}
	if SeverityMedium != "MEDIUM" {
		t.Error("SeverityMedium should be 'MEDIUM'")
	}
	if SeverityLow != "LOW" {
		t.Error("SeverityLow should be 'LOW'")
	}
	if SeverityInfo != "INFO" {
		t.Error("SeverityInfo should be 'INFO'")
	}
}
